package cleanup_app_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/backup_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/app/cleanup_app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/backupstore"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type backupOptions struct{ dir string }

func (o backupOptions) DataDir() string           { return o.dir }
func (o backupOptions) BackupHelperImage() string { return "busybox:1.37" }
func (o backupOptions) CheckshipVersion() string  { return "test" }

// Has one volume and can be made to fail its backup.
type volumesProvider struct {
	domain.Provider
	volumes   []domain.VolumeInfo
	backupErr error
	backups   int
	cleaned   bool
}

func (p *volumesProvider) Volumes(context.Context, domain.Target, domain.AppID, domain.Environment) ([]domain.VolumeInfo, []volumearchive.SkippedMount, error) {
	return p.volumes, nil, nil
}

func (p *volumesProvider) BackupVolumes(_ context.Context, _ domain.Target, _ domain.AppID, _ domain.Environment, opts domain.BackupVolumesOptions, w io.Writer) (domain.BackupVolumesResult, error) {
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
	_ = tw.Close()
	_, _ = aw.AddVolume("dbdata", &buf, "v")
	summary, err := aw.Close()
	return domain.BackupVolumesResult{Manifest: m, Summary: summary}, err
}

func (p *volumesProvider) Cleanup(_ context.Context, _ domain.AppID, _ domain.Target, _ domain.Environment, s domain.CleanupStrategy) error {
	p.cleaned = s != domain.CleanupStrategySkip
	return nil
}

func Test_CleanupAppBackup(t *testing.T) {
	logger := must.Panic(log.NewLogger())

	type env struct {
		ctx     *fixture.Context
		handler func(cleanup_app.Command) error
		cmd     cleanup_app.Command
		app     domain.App
	}

	arrange := func(t *testing.T, provider *volumesProvider) env {
		user := authfixture.User()
		oldTarget := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		oldTarget.Configured(oldTarget.CurrentVersion(), nil, nil)
		newTarget := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(
			domain.NewEnvironmentConfig(oldTarget.ID()),
			domain.NewEnvironmentConfig(oldTarget.ID()),
		))
		// Deployed on the old target, then production moved to the new one
		deployment := fixture.Deployment(fixture.FromApp(app), fixture.ForEnvironment(domain.Production), fixture.WithDeploymentRequestedBy(user.ID()))
		assert.Nil(t, deployment.HasStarted())
		assert.Nil(t, deployment.HasEnded(domain.Services{}, nil))
		assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(newTarget.ID()), true, true)))

		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&oldTarget, &newTarget), fixture.WithApps(&app), fixture.WithDeployments(&deployment))
		files := backupstore.New(backupOptions{t.TempDir()}, nil)
		runner := backup_volumes.NewRunner(ctx.TargetsStore, ctx.BackupsStore, ctx.BackupsStore, provider, files, backupOptions{}, logger)
		handler := cleanup_app.Handler(ctx.TargetsStore, ctx.DeploymentsStore, provider, cleanup_app.Backups{
			Apps: ctx.AppsStore, Reader: ctx.BackupsStore, Restores: ctx.RestoresStore, Runner: runner,
		})

		return env{
			ctx: ctx,
			app: app,
			cmd: cleanup_app.Command{
				TargetID:    string(oldTarget.ID()),
				AppID:       string(app.ID()),
				Environment: string(domain.Production),
				From:        deployment.Requested().At().Add(-time.Hour),
				To:          time.Now().UTC(),
				Backup:      true,
				Trigger:     string(domain.BackupTriggerBeforeTargetChange),
				RequestedBy: string(user.ID()),
			},
			handler: func(cmd cleanup_app.Command) error {
				_, err := handler(ctx.Context, cmd)
				return err
			},
		}
	}

	backups := func(t *testing.T, e env) []domain.Backup {
		return must.Panic(e.ctx.BackupsStore.GetByApp(e.ctx.Context, e.app.ID()))
	}

	t.Run("should back up the volumes on the old target before removing them", func(t *testing.T) {
		provider := &volumesProvider{volumes: []domain.VolumeInfo{{Key: "dbdata"}}}
		e := arrange(t, provider)

		assert.Nil(t, e.handler(e.cmd))

		assert.True(t, provider.cleaned)
		list := backups(t, e)
		assert.HasLength(t, 1, list)
		assert.Equal(t, domain.BackupStatusSucceeded, list[0].Status())
		assert.Equal(t, domain.BackupTriggerBeforeTargetChange, list[0].Trigger())
		assert.Equal(t, domain.TargetID(e.cmd.TargetID), list[0].Target())
	})

	t.Run("should keep the volumes when the backup fails, and reuse it on the next attempt", func(t *testing.T) {
		provider := &volumesProvider{volumes: []domain.VolumeInfo{{Key: "dbdata"}}, backupErr: domain.ErrHelperImagePullFailed}
		e := arrange(t, provider)

		assert.ErrorIs(t, domain.ErrHelperImagePullFailed, e.handler(e.cmd))
		assert.False(t, provider.cleaned)

		provider.backupErr = nil
		assert.Nil(t, e.handler(e.cmd))

		assert.True(t, provider.cleaned)
		assert.Equal(t, 2, provider.backups)
		list := backups(t, e)
		assert.HasLength(t, 1, list)
		assert.Equal(t, domain.BackupStatusSucceeded, list[0].Status())
	})

	t.Run("should queue one restore on the new target when moving the data", func(t *testing.T) {
		provider := &volumesProvider{volumes: []domain.VolumeInfo{{Key: "dbdata"}}}
		e := arrange(t, provider)
		e.cmd.MoveVolumes = true

		assert.Nil(t, e.handler(e.cmd))
		assert.Nil(t, e.handler(e.cmd)) // A retried job does not move the data twice

		list := backups(t, e)
		assert.HasLength(t, 1, list)
		restores := must.Panic(e.ctx.RestoresStore.GetByBackup(e.ctx.Context, list[0].ID()))
		assert.HasLength(t, 1, restores)
		assert.Equal(t, e.app.Config(domain.Production).Target(), restores[0].Target())
		assert.False(t, restores[0].SnapshotFirst())
		assert.Equal(t, 1, provider.backups)
	})

	t.Run("should go on without a backup when there are no volumes", func(t *testing.T) {
		provider := &volumesProvider{}
		e := arrange(t, provider)

		assert.Nil(t, e.handler(e.cmd))

		assert.True(t, provider.cleaned)
		assert.Equal(t, 0, provider.backups)
		assert.HasLength(t, 0, backups(t, e))
	})
}
