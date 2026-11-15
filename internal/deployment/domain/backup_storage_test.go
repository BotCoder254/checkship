package domain_test

import (
	"errors"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_BackupStorage(t *testing.T) {
	t.Run("should keep copies in the data folder by default", func(t *testing.T) {
		s := domain.DefaultBackupStorage()

		access, err := s.Remote()

		assert.Nil(t, err)
		assert.False(t, access.HasValue())
		assert.True(t, s.KeepLocal())
		assert.False(t, s.Saved())
	})

	t.Run("should not be used before the built-in storage is ready", func(t *testing.T) {
		s := domain.DefaultBackupStorage()
		s.UseBuiltin("local", "key", "secret")

		_, err := s.Remote()
		assert.ErrorIs(t, domain.ErrStorageNotReady, err)
		assert.Equal(t, domain.BackupStorageConfiguring, s.Status())

		s.BuiltinConfigured(s.Version(), "checkship-storage:7070", false, nil)

		access, err := s.Remote()
		assert.Nil(t, err)
		assert.Equal(t, "checkship-storage:7070", access.MustGet().Endpoint)
		assert.Equal(t, domain.BuiltinStorageBucket, access.MustGet().Bucket)
		assert.True(t, access.MustGet().PathStyle)
	})

	t.Run("should ignore the outcome of a setup for older settings", func(t *testing.T) {
		s := domain.DefaultBackupStorage()
		s.UseBuiltin("local", "key", "secret")
		old := s.Version()
		s.UseBuiltin("remote", "key", "secret")

		s.BuiltinConfigured(old.Add(-1), "checkship-storage:7070", false, nil)

		assert.Equal(t, domain.BackupStorageConfiguring, s.Status())
	})

	t.Run("should record why the built-in storage failed", func(t *testing.T) {
		s := domain.DefaultBackupStorage()
		s.UseBuiltin("local", "key", "secret")

		s.BuiltinConfigured(s.Version(), "", false, errors.Join(domain.ErrStorageUnreachable))

		assert.Equal(t, domain.BackupStorageFailed, s.Status())
		assert.Equal(t, "storage_unreachable", s.ErrorCode().MustGet())
	})

	t.Run("should keep the keys and move the archives when the built-in storage changes target", func(t *testing.T) {
		s := domain.DefaultBackupStorage()
		s.UseBuiltin("local", "key", "secret")
		s.BuiltinConfigured(s.Version(), "checkship-storage:7070", false, nil)

		s.UseBuiltin("remote", "other", "other-secret")

		assert.Equal(t, "key", s.Access().AccessKey)
		evt := assert.EventIs[domain.BackupStorageChanged](t, &s, 2)
		assert.Equal(t, domain.TargetID("local"), evt.Previous.MustGet())
		assert.Equal(t, "checkship-storage:7070", evt.PreviousAccess.MustGet().Endpoint)
	})

	t.Run("should keep the secret of an external storage when none is given", func(t *testing.T) {
		s := domain.DefaultBackupStorage()
		s.UseExternal(domain.BackupStorageAccess{Endpoint: "https://garage.example.com:3900/", Bucket: "b", AccessKey: "a", SecretKey: "first"})
		s.UseExternal(domain.BackupStorageAccess{Endpoint: "garage.example.com:3900", Bucket: "b", AccessKey: "a"})

		assert.Equal(t, "garage.example.com:3900", s.Access().Endpoint)
		assert.Equal(t, "first", string(s.Access().SecretKey))
		_, err := s.Remote()
		assert.Nil(t, err)
	})
}
