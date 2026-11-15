package sqlite_test

import (
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

func Test_BackupsStore(t *testing.T) {
	user := authfixture.User()
	target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
	config := domain.NewEnvironmentConfig(target.ID())
	app := fixture.App(fixture.WithAppName("blog"), fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(config, config))

	t.Run("should write and read a backup through its whole life", func(t *testing.T) {
		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&target), fixture.WithApps(&app))
		store := ctx.BackupsStore

		b, err := domain.NewBackup(app, domain.Production, domain.BackupTriggerManual, []string{"dbdata"}, true, user.ID())
		assert.Nil(t, err)
		assert.Nil(t, store.Write(ctx.Context, &b))

		got, err := store.GetByID(ctx.Context, b.ID())
		assert.Nil(t, err)
		assert.Equal(t, domain.BackupStatusPending, got.Status())
		assert.ArrayEqual(t, []string{"dbdata"}, got.Volumes())
		assert.True(t, got.StopContainers())
		assert.Equal(t, app.Name(), got.AppName())
		assert.Equal(t, user.ID(), got.Requested().By())

		assert.Nil(t, got.Started())
		got.Succeeded(domain.BackupResult{
			Size:    412,
			SHA256:  "abc",
			Volumes: []volumearchive.VolumeStats{{Key: "dbdata", Size: 400, Files: 2}},
			Skipped: []volumearchive.SkippedMount{{Service: "worker", Kind: "bind", Source: "./cache", Target: "/cache"}},
		}, true, false, apperr.Wrap(domain.ErrStorageUnreachable, nil))
		assert.Nil(t, store.Write(ctx.Context, &got))

		got, err = store.GetByID(ctx.Context, b.ID())
		assert.Nil(t, err)
		assert.Equal(t, domain.BackupStatusSucceeded, got.Status())
		assert.True(t, got.Local())
		assert.False(t, got.Stored())
		assert.Equal(t, "storage_unreachable", got.StoreError().MustGet())
		result := got.Result().MustGet()
		assert.Equal(t, int64(412), result.Size)
		assert.HasLength(t, 1, result.Skipped)

		assert.Nil(t, got.HasCopies(true, true, nil))
		assert.Nil(t, store.Write(ctx.Context, &got))
		got, _ = store.GetByID(ctx.Context, b.ID())
		assert.True(t, got.Stored())
		assert.False(t, got.StoreError().HasValue())

		list, err := store.GetByApp(ctx.Context, app.ID())
		assert.Nil(t, err)
		assert.HasLength(t, 1, list)

		assert.Nil(t, got.Delete())
		assert.Nil(t, store.Write(ctx.Context, &got))
		_, err = store.GetByID(ctx.Context, b.ID())
		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should record a failure", func(t *testing.T) {
		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&target), fixture.WithApps(&app))
		b, _ := domain.NewBackup(app, domain.Staging, domain.BackupTriggerManual, nil, false, user.ID())
		b.Failed(domain.ErrHelperImagePullFailed)
		assert.Nil(t, ctx.BackupsStore.Write(ctx.Context, &b))

		got, err := ctx.BackupsStore.GetByID(ctx.Context, b.ID())
		assert.Nil(t, err)
		assert.Equal(t, domain.BackupStatusFailed, got.Status())
		assert.Equal(t, "helper_image_pull_failed", got.ErrorCode().MustGet())
		assert.HasLength(t, 0, got.Volumes())
	})
}
