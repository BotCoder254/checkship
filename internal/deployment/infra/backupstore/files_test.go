package backupstore_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/backupstore"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

type dataDir string

func (d dataDir) DataDir() string { return string(d) }

type memoryRemote struct {
	mu      sync.Mutex
	objects map[string][]byte
	putErr  error
}

func (m *memoryRemote) Put(_ context.Context, key string, r io.Reader, size int64) error {
	if m.putErr != nil {
		return m.putErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) != size {
		return errors.New("size mismatch")
	}
	m.mu.Lock()
	m.objects[key] = b
	m.mu.Unlock()
	return nil
}

func (m *memoryRemote) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objects[key]
	if !ok {
		return nil, 0, errors.New("NoSuchKey")
	}
	return io.NopCloser(bytes.NewReader(b)), int64(len(b)), nil
}

func (m *memoryRemote) Remove(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.objects, key)
	m.mu.Unlock()
	return nil
}

func Test_Files(t *testing.T) {
	ctx := context.Background()
	newBackup := func() domain.Backup {
		b, _ := domain.NewBackup(fixture.App(), domain.Production, domain.BackupTriggerManual, nil, true, "uid")
		return b
	}
	write := func(t *testing.T, files domain.BackupFiles, b domain.Backup, content string) {
		f, err := files.Create(ctx)
		assert.Nil(t, err)
		_, err = f.Write([]byte(content))
		assert.Nil(t, err)
		assert.Nil(t, f.Commit(b))
		assert.Nil(t, f.Abort()) // no-op after commit
	}
	read := func(t *testing.T, files domain.BackupFiles, b domain.Backup) string {
		rc, size, err := files.Open(ctx, b)
		assert.Nil(t, err)
		defer rc.Close()
		data, _ := io.ReadAll(rc)
		assert.Equal(t, int64(len(data)), size)
		return string(data)
	}

	t.Run("should keep archives in the data folder, readable by checkship only", func(t *testing.T) {
		dir := t.TempDir()
		files := backupstore.New(dataDir(dir), nil)
		b := newBackup()

		write(t, files, b, "archive")
		b.Succeeded(domain.BackupResult{}, true, false, nil)

		path := filepath.Join(dir, "backups", string(b.App()), string(b.ID())+".tar.gz")
		info, err := os.Stat(path)
		assert.Nil(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		assert.Equal(t, "archive", read(t, files, b))

		result, err := files.Store(ctx, b)
		assert.Nil(t, err)
		assert.False(t, result.Stored)
		assert.True(t, result.KeepLocal)

		assert.Nil(t, files.Delete(ctx, b))
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("should leave nothing behind when an archive is aborted", func(t *testing.T) {
		dir := t.TempDir()
		files := backupstore.New(dataDir(dir), nil)

		f, err := files.Create(ctx)
		assert.Nil(t, err)
		_, _ = f.Write([]byte("half"))
		assert.Nil(t, f.Abort())

		entries, _ := os.ReadDir(filepath.Join(dir, "backups", ".partial"))
		assert.HasLength(t, 0, entries)
	})

	t.Run("should copy to the storage and read from it once the local copy is gone", func(t *testing.T) {
		remote := &memoryRemote{objects: map[string][]byte{}}
		files := backupstore.New(dataDir(t.TempDir()), func(context.Context) (backupstore.Remote, bool, error) { return remote, false, nil })
		b := newBackup()
		write(t, files, b, "archive")

		result, err := files.Store(ctx, b)
		assert.Nil(t, err)
		assert.True(t, result.Stored)
		assert.False(t, result.KeepLocal)
		assert.Equal(t, "archive", string(remote.objects[backupstore.Key(b)]))

		assert.Nil(t, files.RemoveLocal(ctx, b))
		b.Succeeded(domain.BackupResult{}, false, true, nil)
		assert.Equal(t, "archive", read(t, files, b))

		assert.Nil(t, files.Delete(ctx, b))
		assert.HasLength(t, 0, keys(remote))
	})

	t.Run("should report the storage as unreachable and keep the local copy", func(t *testing.T) {
		remote := &memoryRemote{objects: map[string][]byte{}, putErr: errors.New("connection refused")}
		files := backupstore.New(dataDir(t.TempDir()), func(context.Context) (backupstore.Remote, bool, error) { return remote, false, nil })
		b := newBackup()
		write(t, files, b, "archive")

		result, err := files.Store(ctx, b)

		assert.ErrorIs(t, domain.ErrStorageUnreachable, err)
		assert.True(t, result.KeepLocal)
		assert.False(t, result.Stored)
	})
}

func keys(m *memoryRemote) []string {
	out := []string{}
	for k := range m.objects {
		out = append(out, k)
	}
	return out
}
