package domain_test

import (
	"errors"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

func Test_Backup(t *testing.T) {
	result := domain.BackupResult{
		Size:    412,
		SHA256:  "abc",
		Volumes: []volumearchive.VolumeStats{{Key: "dbdata", Size: 400, Files: 3}},
	}

	t.Run("should be requested on the target of the environment", func(t *testing.T) {
		app := fixture.App(fixture.WithEnvironmentConfig(domain.NewEnvironmentConfig("prod-target"), domain.NewEnvironmentConfig("stg-target")))

		b, err := domain.NewBackup(app, domain.Staging, domain.BackupTriggerManual, []string{"dbdata"}, true, "uid")

		assert.Nil(t, err)
		assert.NotZero(t, b.ID())
		assert.Equal(t, domain.BackupStatusPending, b.Status())
		assert.Equal(t, domain.TargetID("stg-target"), b.Target())

		requested := assert.EventIs[domain.BackupRequested](t, &b, 0)
		assert.Equal(t, app.ID(), requested.App)
		assert.Equal(t, app.Name(), requested.AppName)
		assert.Equal(t, domain.Staging, requested.Environment)
		assert.True(t, requested.StopContainers)
		assert.ArrayEqual(t, []string{"dbdata"}, requested.Volumes)
	})

	t.Run("should run on an explicit target for an environment that is moving", func(t *testing.T) {
		app := fixture.App()

		b, err := domain.NewBackupOnTarget(app, domain.Production, "old-target", domain.BackupTriggerBeforeTargetChange, nil, true, "uid")

		assert.Nil(t, err)
		assert.Equal(t, domain.TargetID("old-target"), assert.EventIs[domain.BackupRequested](t, &b, 0).Target)
	})

	t.Run("should refuse invalid triggers, environments and volume keys", func(t *testing.T) {
		app := fixture.App()

		_, err := domain.NewBackup(app, domain.Production, "nope", nil, true, "uid")
		assert.ErrorIs(t, domain.ErrInvalidBackupTrigger, err)

		_, err = domain.NewBackup(app, domain.Production, domain.BackupTriggerUpload, nil, true, "uid")
		assert.ErrorIs(t, domain.ErrInvalidBackupTrigger, err)

		_, err = domain.NewBackup(app, "dev", domain.BackupTriggerManual, nil, true, "uid")
		assert.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)

		_, err = domain.NewBackup(app, domain.Production, domain.BackupTriggerManual, []string{"../etc"}, true, "uid")
		assert.ErrorIs(t, volumearchive.ErrInvalidVolumeKey, err)
	})

	t.Run("should only allow a before_delete backup once the app is being deleted", func(t *testing.T) {
		app := fixture.App()
		app.RequestCleanup("uid")

		_, err := domain.NewBackup(app, domain.Production, domain.BackupTriggerManual, nil, true, "uid")
		assert.ErrorIs(t, domain.ErrAppCleanupRequested, err)

		_, err = domain.NewBackup(app, domain.Production, domain.BackupTriggerBeforeDelete, nil, true, "uid")
		assert.Nil(t, err)
	})

	t.Run("should go through running then succeeded", func(t *testing.T) {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")

		assert.Nil(t, b.Started())
		assert.Nil(t, b.Started()) // A retried job records nothing more
		assert.Equal(t, domain.BackupStatusRunning, b.Status())
		assert.HasNEvents(t, 2, &b)

		b.Succeeded(result, true, true, nil)

		succeeded := assert.EventIs[domain.BackupSucceeded](t, &b, 2)
		assert.Equal(t, domain.BackupStatusSucceeded, b.Status())
		assert.True(t, succeeded.Local)
		assert.True(t, succeeded.Stored)
		assert.False(t, succeeded.StoreError.HasValue())
		assert.DeepEqual(t, result, b.Result().MustGet())
	})

	t.Run("should keep the local copy and the reason when the storage copy failed", func(t *testing.T) {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")

		b.Succeeded(result, false, false, apperr.Wrap(domain.ErrStorageUnreachable, errors.New("dial tcp: refused")))

		assert.True(t, b.Local())
		assert.False(t, b.Stored())
		assert.Equal(t, "storage_unreachable", b.StoreError().MustGet())
	})

	t.Run("should record the error code of a failure, never the raw error", func(t *testing.T) {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")

		b.Failed(apperr.Wrap(domain.ErrHelperImagePullFailed, errors.New("toomanyrequests")))
		assert.Equal(t, "helper_image_pull_failed", b.ErrorCode().MustGet())

		b2, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")
		b2.Failed(errors.New("secret details"))
		assert.Equal(t, "unexpected_error", b2.ErrorCode().MustGet())
		assert.Equal(t, domain.BackupStatusFailed, b2.Status())
	})

	t.Run("should never drop its last copy", func(t *testing.T) {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")
		b.Succeeded(result, true, false, nil)

		assert.Nil(t, b.HasCopies(false, false, nil))
		assert.True(t, b.Local())

		assert.Nil(t, b.HasCopies(false, true, nil))
		assert.False(t, b.Local())
		assert.True(t, b.Stored())
		assert.EventIs[domain.BackupLocationsChanged](t, &b, 2)
	})

	t.Run("should only be restorable once succeeded", func(t *testing.T) {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")

		_, err := b.Restorable()
		assert.ErrorIs(t, domain.ErrBackupNotRestorable, err)

		b.Succeeded(result, true, false, nil)
		got, err := b.Restorable()
		assert.Nil(t, err)
		assert.Equal(t, "abc", got.SHA256)
	})

	t.Run("should not be deleted while its job runs", func(t *testing.T) {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")

		assert.ErrorIs(t, domain.ErrBackupNotFinished, b.Delete())

		b.Succeeded(result, true, true, nil)
		assert.Nil(t, b.Delete())
		deleted := assert.EventIs[domain.BackupDeleted](t, &b, 2)
		assert.True(t, deleted.Local)
		assert.True(t, deleted.Stored)
	})

	t.Run("should record an uploaded archive as succeeded", func(t *testing.T) {
		app := fixture.App()

		b, err := domain.NewUploadedBackup(app, domain.Production, result, "uid")

		assert.Nil(t, err)
		assert.Equal(t, domain.BackupTriggerUpload, b.Trigger())
		assert.Equal(t, domain.BackupStatusSucceeded, b.Status())
		assert.True(t, b.Local())
		assert.HasNEvents(t, 2, &b)
	})
}
