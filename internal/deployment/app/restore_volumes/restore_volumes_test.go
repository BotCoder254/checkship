package restore_volumes_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/backup_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/app/restore_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/backupstore"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type options struct{ dir string }

func (o options) DataDir() string           { return o.dir }
func (o options) BackupHelperImage() string { return "busybox:1.37" }
func (o options) CheckshipVersion() string  { return "test" }

type fakeProvider struct {
	domain.Provider
	backupErr   error
	restored    map[string]string // volume key -> PG_VERSION content restored
	restoreOpts domain.RestoreVolumesOptions
	backups     int
}

func (p *fakeProvider) Volumes(context.Context, domain.Target, domain.AppID, domain.Environment) ([]domain.VolumeInfo, []volumearchive.SkippedMount, error) {
	return []domain.VolumeInfo{{Key: "dbdata"}}, nil, nil
}

func (p *fakeProvider) BackupVolumes(_ context.Context, _ domain.Target, _ domain.AppID, _ domain.Environment, opts domain.BackupVolumesOptions, w io.Writer) (domain.BackupVolumesResult, error) {
	p.backups++
	if p.backupErr != nil {
		return domain.BackupVolumesResult{}, p.backupErr
	}
	m := opts.Manifest
	m.Volumes = []volumearchive.ManifestVolume{{Key: "dbdata"}}
	aw := must.Panic(volumearchive.NewWriter(w, m))
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "v", Typeflag: tar.TypeDir, Mode: 0o755})
	_ = tw.WriteHeader(&tar.Header{Name: "v/PG_VERSION", Typeflag: tar.TypeReg, Mode: 0o644, Size: 2})
	_, _ = tw.Write([]byte("17"))
	_ = tw.Close()
	_, _ = aw.AddVolume("dbdata", &buf, "v")
	summary, err := aw.Close()
	return domain.BackupVolumesResult{Manifest: m, Summary: summary}, err
}

func (p *fakeProvider) RestoreVolumes(_ context.Context, _ domain.Target, _ domain.AppID, _ domain.Environment, opts domain.RestoreVolumesOptions, r io.Reader) ([]string, error) {
	p.restoreOpts = opts
	p.restored = map[string]string{}
	archive, err := volumearchive.Open(r)
	if err != nil {
		return nil, err
	}
	var keys []string
	err = archive.Volumes(nil, func(key string, _ *volumearchive.RootInfo, content io.Reader) error {
		tr := tar.NewReader(content)
		for {
			hdr, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			if hdr.Name == "PG_VERSION" {
				p.restored[key] = string(must.Panic(io.ReadAll(tr)))
			}
		}
		keys = append(keys, key)
		return nil
	})
	return keys, err
}

func Test_RestoreVolumes(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	user := authfixture.User()
	target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
	config := domain.NewEnvironmentConfig(target.ID())
	app := fixture.App(fixture.WithAppName("blog"), fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(config, config))

	type env struct {
		ctx      *fixture.Context
		provider *fakeProvider
		files    domain.BackupFiles
		dir      string
		backup   domain.Backup
		run      func(domain.Restore) domain.Restore
	}

	arrange := func(t *testing.T) env {
		dir := t.TempDir()
		provider := &fakeProvider{}
		backup := must.Panic(domain.NewBackup(app, domain.Production, domain.BackupTriggerManual, nil, true, user.ID()))
		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&target), fixture.WithApps(&app), fixture.WithBackups(&backup))
		files := backupstore.New(options{dir}, nil)
		runner := backup_volumes.NewRunner(ctx.TargetsStore, ctx.BackupsStore, ctx.BackupsStore, provider, files, options{}, logger)

		// A real archive to restore from
		stored, err := runner.Run(ctx.Context, must.Panic(ctx.BackupsStore.GetByID(ctx.Context, backup.ID())))
		assert.Nil(t, err)

		handler := restore_volumes.Handler(ctx.RestoresStore, ctx.RestoresStore, ctx.BackupsStore,
			ctx.AppsStore, ctx.TargetsStore, files, provider, runner, options{}, logger)

		return env{ctx, provider, files, dir, stored, func(r domain.Restore) domain.Restore {
			assert.Nil(t, ctx.RestoresStore.Write(ctx.Context, &r))
			_, err := handler(ctx.Context, restore_volumes.Command{ID: string(r.ID())})
			assert.Nil(t, err)
			return must.Panic(ctx.RestoresStore.GetByID(ctx.Context, r.ID()))
		}}
	}

	t.Run("should back up the destination first, then restore the archive into staging", func(t *testing.T) {
		e := arrange(t)
		restore := must.Panic(domain.NewRestore(e.backup, app, domain.Staging, nil, true, user.ID()))

		got := e.run(restore)

		assert.Equal(t, domain.RestoreStatusSucceeded, got.Status())
		assert.ArrayEqual(t, []string{"dbdata"}, got.Restored())
		assert.Equal(t, "17", e.provider.restored["dbdata"])
		assert.Equal(t, "blog-staging-"+lower(string(app.ID())), e.provider.restoreOpts.ProjectName)
		assert.Equal(t, 2, e.provider.backups) // the backup itself, then the snapshot

		snapshot := must.Panic(e.ctx.BackupsStore.GetByID(e.ctx.Context, got.Snapshot().MustGet()))
		assert.Equal(t, domain.BackupTriggerBeforeRestore, snapshot.Trigger())
		assert.Equal(t, domain.Staging, snapshot.Environment())
		assert.Equal(t, domain.BackupStatusSucceeded, snapshot.Status())
	})

	t.Run("should refuse an archive changed since the backup and touch nothing", func(t *testing.T) {
		e := arrange(t)
		path := filepath.Join(e.dir, "backups", filepath.FromSlash(backupstore.Key(e.backup)))
		data := must.Panic(os.ReadFile(path))
		data[len(data)-1] ^= 0xff
		assert.Nil(t, os.WriteFile(path, data, 0o600))

		got := e.run(must.Panic(domain.NewRestore(e.backup, app, domain.Production, nil, true, user.ID())))

		assert.Equal(t, domain.RestoreStatusFailed, got.Status())
		assert.Equal(t, "checksum_mismatch", got.ErrorCode().MustGet())
		assert.Equal(t, 1, e.provider.backups)
		assert.True(t, e.provider.restored == nil)
	})

	t.Run("should not replace data whose snapshot failed", func(t *testing.T) {
		e := arrange(t)
		e.provider.backupErr = domain.ErrHelperImagePullFailed

		got := e.run(must.Panic(domain.NewRestore(e.backup, app, domain.Production, nil, true, user.ID())))

		assert.Equal(t, domain.RestoreStatusFailed, got.Status())
		assert.Equal(t, "helper_image_pull_failed", got.ErrorCode().MustGet())
		assert.True(t, got.Snapshot().HasValue())
		assert.True(t, e.provider.restored == nil)
	})

	t.Run("should skip the snapshot when asked", func(t *testing.T) {
		e := arrange(t)

		got := e.run(must.Panic(domain.NewRestore(e.backup, app, domain.Production, []string{"dbdata"}, false, user.ID())))

		assert.Equal(t, domain.RestoreStatusSucceeded, got.Status())
		assert.False(t, got.Snapshot().HasValue())
		assert.Equal(t, 1, e.provider.backups)
		assert.ArrayEqual(t, []string{"dbdata"}, e.provider.restoreOpts.Volumes)
	})
}

func lower(s string) string { return string(bytes.ToLower([]byte(s))) }
