package backup_volumes_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/backup_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/backupstore"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type options struct{ dir string }

func (o options) DataDir() string           { return o.dir }
func (o options) BackupHelperImage() string { return "busybox:1.37" }
func (o options) CheckshipVersion() string  { return "test" }

// Writes a real archive with one volume, or fails like a target would.
type fakeProvider struct {
	domain.Provider
	err  error
	opts domain.BackupVolumesOptions
}

func (p *fakeProvider) BackupVolumes(_ context.Context, _ domain.Target, _ domain.AppID, _ domain.Environment, opts domain.BackupVolumesOptions, w io.Writer) (domain.BackupVolumesResult, error) {
	p.opts = opts
	if p.err != nil {
		return domain.BackupVolumesResult{}, p.err
	}
	m := opts.Manifest
	m.Volumes = []volumearchive.ManifestVolume{{Key: "dbdata"}}
	aw, err := volumearchive.NewWriter(w, m)
	if err != nil {
		return domain.BackupVolumesResult{}, err
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "v", Typeflag: tar.TypeDir, Mode: 0o755})
	_ = tw.WriteHeader(&tar.Header{Name: "v/PG_VERSION", Typeflag: tar.TypeReg, Mode: 0o644, Size: 2})
	_, _ = tw.Write([]byte("17"))
	_ = tw.Close()
	if _, err = aw.AddVolume("dbdata", &buf, "v"); err != nil {
		return domain.BackupVolumesResult{}, err
	}
	summary, err := aw.Close()
	return domain.BackupVolumesResult{Manifest: m, Summary: summary}, err
}

func Test_BackupVolumes(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	user := authfixture.User()
	target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
	config := domain.NewEnvironmentConfig(target.ID())
	app := fixture.App(fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(config, config))

	arrange := func(t *testing.T, provider *fakeProvider) (*fixture.Context, bus.RequestHandler[bus.UnitType, backup_volumes.Command], domain.Backup, domain.BackupFiles) {
		backup := must.Panic(domain.NewBackup(app, domain.Production, domain.BackupTriggerManual, []string{"dbdata"}, true, user.ID()))
		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&target), fixture.WithApps(&app), fixture.WithBackups(&backup))
		files := backupstore.New(options{t.TempDir()}, nil)
		runner := backup_volumes.NewRunner(ctx.TargetsStore, ctx.BackupsStore, ctx.BackupsStore, provider, files, options{}, logger)
		return ctx, backup_volumes.Handler(ctx.BackupsStore, runner), backup, files
	}

	t.Run("should write the archive and record what it holds", func(t *testing.T) {
		provider := &fakeProvider{}
		ctx, handler, backup, files := arrange(t, provider)

		_, err := handler(ctx.Context, backup_volumes.Command{ID: string(backup.ID())})

		assert.Nil(t, err)
		assert.True(t, provider.opts.StopContainers)
		assert.ArrayEqual(t, []string{"dbdata"}, provider.opts.Volumes)
		assert.Equal(t, "test", provider.opts.Manifest.Checkship)
		assert.Equal(t, string(app.ID()), provider.opts.Manifest.AppID)

		got := must.Panic(ctx.BackupsStore.GetByID(ctx.Context, backup.ID()))
		assert.Equal(t, domain.BackupStatusSucceeded, got.Status())
		assert.True(t, got.Local())
		assert.False(t, got.Stored())
		result := got.Result().MustGet()
		assert.HasLength(t, 1, result.Volumes)
		assert.Equal(t, int64(2), result.Volumes[0].Size)

		rc, size, err := files.Open(ctx.Context, got)
		assert.Nil(t, err)
		defer rc.Close()
		assert.Equal(t, result.Size, size)
		archive := must.Panic(volumearchive.Open(rc))
		assert.Equal(t, "dbdata", archive.Manifest().Volumes[0].Key)
	})

	t.Run("should record a failure with its code and not ask for a retry", func(t *testing.T) {
		provider := &fakeProvider{err: errors.Join(domain.ErrNoVolumes)}
		ctx, handler, backup, _ := arrange(t, provider)

		_, err := handler(ctx.Context, backup_volumes.Command{ID: string(backup.ID())})

		assert.Nil(t, err)
		got := must.Panic(ctx.BackupsStore.GetByID(ctx.Context, backup.ID()))
		assert.Equal(t, domain.BackupStatusFailed, got.Status())
		assert.Equal(t, "no_volumes", got.ErrorCode().MustGet())
	})

	t.Run("should do nothing for a backup already done or gone", func(t *testing.T) {
		provider := &fakeProvider{err: errors.New("must not be called")}
		ctx, handler, backup, _ := arrange(t, provider)
		stored := must.Panic(ctx.BackupsStore.GetByID(ctx.Context, backup.ID()))
		stored.Failed(domain.ErrNoVolumes)
		assert.Nil(t, ctx.BackupsStore.Write(ctx.Context, &stored))

		_, err := handler(ctx.Context, backup_volumes.Command{ID: string(backup.ID())})
		assert.Nil(t, err)
		_, err = handler(ctx.Context, backup_volumes.Command{ID: "unknown"})
		assert.Nil(t, err)
		assert.HasLength(t, 0, provider.opts.Volumes)
	})
}
