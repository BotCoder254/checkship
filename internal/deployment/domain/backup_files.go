package domain

import (
	"context"
	"io"
)

type (
	// Where backup archives are kept: checkship's data folder, plus the storage
	// configured for backups when there is one.
	BackupFiles interface {
		// Starts a new archive in the data folder. Nothing is visible until Commit.
		Create(context.Context) (BackupFile, error)
		// Copies the local archive of a backup to the storage. Stored is false
		// with a nil error when no storage is configured.
		Store(context.Context, Backup) (BackupStoreResult, error)
		// Removes the copy in the data folder.
		RemoveLocal(context.Context, Backup) error
		// Opens a copy for reading: the local one first, the storage otherwise.
		Open(context.Context, Backup) (io.ReadCloser, int64, error)
		// Removes every copy of a backup.
		Delete(context.Context, Backup) error
	}

	BackupFile interface {
		io.Writer
		// Moves the archive to its place for the given backup.
		Commit(Backup) error
		// Throws the archive away. Safe to call after Commit.
		Abort() error
	}

	BackupStoreResult struct {
		Stored    bool // A copy is now in the storage
		KeepLocal bool // The storage settings ask to keep the copy in the data folder
	}
)
