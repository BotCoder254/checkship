// Package backupstore keeps backup archives in checkship's data folder and,
// when one is configured, in an S3-compatible storage.
package backupstore

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/ostools"
)

const (
	backupsDir   = "backups"
	partialDir   = ".partial"
	archiveExt   = ".tar.gz"
	partialExt   = ".partial"
	filePerm     = 0o600 // Archives hold application data, only checkship reads them
	directoryDir = 0o700
)

type (
	Options interface {
		DataDir() string
	}

	// An S3-compatible place to copy archives to.
	Remote interface {
		Put(ctx context.Context, key string, r io.Reader, size int64) error
		Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
		Remove(ctx context.Context, key string) error
	}

	// Returns the storage currently configured, if any, and whether local
	// copies are kept once uploaded.
	RemoteResolver func(context.Context) (remote Remote, keepLocal bool, err error)

	files struct {
		root    string
		resolve RemoteResolver
	}

	localFile struct {
		f    *os.File
		root string
		done bool
	}
)

// Creates the backup files service. resolve may be nil when there is no storage.
func New(opts Options, resolve RemoteResolver) domain.BackupFiles {
	if resolve == nil {
		resolve = func(context.Context) (Remote, bool, error) { return nil, true, nil }
	}

	return &files{
		root:    filepath.Join(opts.DataDir(), backupsDir),
		resolve: resolve,
	}
}

// Key of a backup archive, the same in the data folder and in the storage.
func Key(b domain.Backup) string {
	return string(b.App()) + "/" + string(b.ID()) + archiveExt
}

func (s *files) localPath(b domain.Backup) string {
	return filepath.Join(s.root, filepath.FromSlash(Key(b)))
}

func (s *files) Create(context.Context) (domain.BackupFile, error) {
	dir := filepath.Join(s.root, partialDir)

	if err := os.MkdirAll(dir, directoryDir); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filepath.Join(dir, id.New[string]()+partialExt), os.O_CREATE|os.O_EXCL|os.O_WRONLY, filePerm)

	if err != nil {
		return nil, err
	}

	return &localFile{f: f, root: s.root}, nil
}

func (s *files) Store(ctx context.Context, b domain.Backup) (domain.BackupStoreResult, error) {
	remote, keepLocal, err := s.resolve(ctx)

	if err != nil {
		return domain.BackupStoreResult{KeepLocal: true}, apperr.Wrap(domain.ErrStorageUnreachable, err)
	}

	if remote == nil {
		return domain.BackupStoreResult{KeepLocal: true}, nil
	}

	f, err := os.Open(s.localPath(b))

	if err != nil {
		return domain.BackupStoreResult{KeepLocal: true}, err
	}

	defer f.Close()

	info, err := f.Stat()

	if err != nil {
		return domain.BackupStoreResult{KeepLocal: true}, err
	}

	if err = remote.Put(ctx, Key(b), f, info.Size()); err != nil {
		return domain.BackupStoreResult{KeepLocal: true}, apperr.Wrap(domain.ErrStorageUnreachable, err)
	}

	return domain.BackupStoreResult{Stored: true, KeepLocal: keepLocal}, nil
}

func (s *files) RemoveLocal(_ context.Context, b domain.Backup) error {
	err := os.Remove(s.localPath(b))

	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}

func (s *files) Open(ctx context.Context, b domain.Backup) (io.ReadCloser, int64, error) {
	if b.Local() {
		f, err := os.Open(s.localPath(b))

		if err == nil {
			info, err := f.Stat()

			if err != nil {
				f.Close()
				return nil, 0, err
			}

			return f, info.Size(), nil
		}

		if !errors.Is(err, fs.ErrNotExist) || !b.Stored() {
			return nil, 0, err
		}
	}

	if !b.Stored() {
		return nil, 0, apperr.ErrNotFound
	}

	remote, _, err := s.resolve(ctx)

	if err != nil {
		return nil, 0, apperr.Wrap(domain.ErrStorageUnreachable, err)
	}

	if remote == nil {
		return nil, 0, domain.ErrStorageUnreachable
	}

	rc, size, err := remote.Get(ctx, Key(b))

	if err != nil {
		return nil, 0, apperr.Wrap(domain.ErrStorageUnreachable, err)
	}

	return rc, size, nil
}

func (s *files) Delete(ctx context.Context, b domain.Backup) error {
	var errs []error

	if err := s.RemoveLocal(ctx, b); err != nil {
		errs = append(errs, err)
	}

	if b.Stored() {
		remote, _, err := s.resolve(ctx)

		switch {
		case err != nil:
			errs = append(errs, apperr.Wrap(domain.ErrStorageUnreachable, err))
		case remote != nil:
			if err = remote.Remove(ctx, Key(b)); err != nil {
				errs = append(errs, apperr.Wrap(domain.ErrStorageUnreachable, err))
			}
		}
	}

	return errors.Join(errs...)
}

func (f *localFile) Write(p []byte) (int, error) { return f.f.Write(p) }

func (f *localFile) Commit(b domain.Backup) error {
	if err := f.f.Sync(); err != nil {
		return err
	}

	if err := f.f.Close(); err != nil {
		return err
	}

	target := filepath.Join(f.root, filepath.FromSlash(Key(b)))

	if err := ostools.MkdirAll(filepath.Dir(target)); err != nil {
		return err
	}

	if err := os.Rename(f.f.Name(), target); err != nil {
		return err
	}

	f.done = true

	return nil
}

func (f *localFile) Abort() error {
	if f.done {
		return nil
	}

	f.done = true
	_ = f.f.Close()

	err := os.Remove(f.f.Name())

	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}
