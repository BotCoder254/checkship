package upload_backup

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

var ErrArchiveTooLarge = apperr.New("archive_too_large")

type (
	// Add a backup archive made by checkship, from this server or another one.
	Command struct {
		bus.Command[string]

		AppID       string                `json:"-"`
		Environment string                `form:"environment"` // The one in the manifest if empty
		Archive     *multipart.FileHeader `form:"archive"`
	}

	Options interface {
		BackupsMaxUploadSize() int64
	}
)

func (Command) Name_() string { return "deployment.command.upload_backup" }

func Handler(
	apps domain.AppsReader,
	writer domain.BackupsWriter,
	files domain.BackupFiles,
	options Options,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		if cmd.Archive == nil {
			return "", validate.NewError(validate.FieldErrors{"archive": domain.ErrInvalidSourcePayload})
		}

		if cmd.Archive.Size > options.BackupsMaxUploadSize() {
			return "", ErrArchiveTooLarge
		}

		app, err := apps.GetByID(ctx, domain.AppID(cmd.AppID))

		if err != nil {
			return "", err
		}

		src, err := cmd.Archive.Open()

		if err != nil {
			return "", err
		}

		defer src.Close()

		file, err := files.Create(ctx)

		if err != nil {
			return "", err
		}

		defer file.Abort()

		manifest, result, err := Import(io.LimitReader(src, options.BackupsMaxUploadSize()+1), file, options.BackupsMaxUploadSize())

		if err != nil {
			return "", err
		}

		env := domain.Environment(manifest.Environment)

		if cmd.Environment != "" {
			env = domain.Environment(cmd.Environment)
		}

		backup, err := domain.NewUploadedBackup(app, env, result, auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return "", err
		}

		if err = file.Commit(backup); err != nil {
			return "", err
		}

		if err = writer.Write(ctx, &backup); err != nil {
			return "", err
		}

		return string(backup.ID()), nil
	}
}

// Copies an archive to dst while checking all of it: the manifest, every entry,
// and the size. Returns what a backup records about it.
func Import(src io.Reader, dst io.Writer, maxSize int64) (volumearchive.Manifest, domain.BackupResult, error) {
	var (
		counter = &countingWriter{}
		hash    = sha256.New()
		tee     = io.TeeReader(src, io.MultiWriter(dst, hash, counter))
		result  domain.BackupResult
	)

	archive, err := volumearchive.Open(tee)

	if err != nil {
		return volumearchive.Manifest{}, result, err
	}

	defer archive.Close()

	manifest := archive.Manifest()

	err = archive.Volumes(nil, func(key string, _ *volumearchive.RootInfo, content io.Reader) error {
		stats := volumearchive.VolumeStats{Key: key}
		tr := tar.NewReader(content)

		for {
			hdr, err := tr.Next()

			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				return err
			}

			stats.Files++

			if hdr.Typeflag == tar.TypeReg {
				n, err := io.Copy(io.Discard, tr)

				if err != nil {
					return err
				}

				stats.Size += n
			}
		}

		result.Volumes = append(result.Volumes, stats)
		return nil
	})

	if err != nil {
		return manifest, result, normalize(err)
	}

	// Whatever follows the tar end (gzip footer, padding) is part of the file
	if _, err = io.Copy(io.Discard, tee); err != nil {
		return manifest, result, err
	}

	if counter.n > maxSize {
		return manifest, result, ErrArchiveTooLarge
	}

	result.Size = counter.n
	result.SHA256 = hex.EncodeToString(hash.Sum(nil))
	result.Skipped = manifest.Skipped

	return manifest, result, nil
}

func normalize(err error) error {
	if _, ok := apperr.As[apperr.Error](err); ok {
		return err
	}

	return errors.Join(volumearchive.ErrInvalidEntry, err)
}

type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}
