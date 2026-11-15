package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/ostools"
	"github.com/checkship-deploy/checkship/pkg/types"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

const (
	tmpPattern        = "checkship-archive-"
	supportedMimeType = "application/x-gzip"
)

var (
	ErrOpenArchiveFailed    = errors.New("open_archive_failed")
	ErrUnzipArchiveFailed   = errors.New("unzip_archive_failed")
	ErrReadArchiveFailed    = errors.New("read_archive_failed")
	ErrWriteFileFailed      = errors.New("write_archive_file_failed")
	ErrWriteDirectoryFailed = errors.New("write_archive_directory_failed")

	ErrInvalidFileType     = apperr.New("invalid_file_type")
	ErrMaxFileSizeExceeded = apperr.New("max_file_size_exceeded")
	ErrInvalidArchiveEntry = apperr.New("invalid_archive_entry") // An entry that would land outside the build directory
)

type (
	Options interface {
		MaxDeploymentArchiveFileSize() int64
	}

	service struct {
		options Options
	}
)

func New(options Options) source.Source {
	return &service{options: options}
}

func (*service) CanPrepare(payload any) bool          { return types.Is[*multipart.FileHeader](payload) }
func (*service) CanFetch(meta domain.SourceData) bool { return types.Is[Data](meta) }

func (t *service) Prepare(ctx context.Context, app domain.App, payload any) (domain.SourceData, error) {
	file, ok := payload.(*multipart.FileHeader)

	if !ok {
		return nil, domain.ErrInvalidSourcePayload
	}

	tmpfile, err := os.CreateTemp("", tmpPattern)

	if err != nil {
		return nil, err
	}

	defer tmpfile.Close()

	// Check the size and type before anything is copied
	archive, err := t.tryOpenFile(file)

	if err != nil {
		os.Remove(tmpfile.Name())
		return nil, validate.Wrap(err, "archive")
	}

	defer archive.Close()

	if _, err := io.Copy(tmpfile, archive); err != nil {
		return nil, err
	}

	return Data(tmpfile.Name()), nil
}

func (t *service) Fetch(ctx context.Context, deploymentCtx domain.DeploymentContext, depl domain.Deployment) error {
	data, ok := depl.Source().(Data)

	if !ok {
		return domain.ErrInvalidSourcePayload
	}

	dir := deploymentCtx.BuildDirectory()
	logger := deploymentCtx.Logger()

	logger.Stepf("extracting archive %s into %s", data, dir)

	// Open the archive file stored in a temporary location
	archive, err := os.Open(string(data))

	if err != nil {
		logger.Error(err)
		return ErrOpenArchiveFailed
	}

	defer archive.Close()

	// And now, uncompress and untar it in the app directory
	gzr, err := gzip.NewReader(archive)

	if err != nil {
		logger.Error(err)
		return ErrUnzipArchiveFailed
	}

	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			logger.Error(err)
			return ErrReadArchiveFailed

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// Where the entry goes, refused if that is outside the build directory
		target, err := safeJoin(dir, header.Name)

		if err != nil {
			logger.Error(fmt.Errorf("archive entry %q: %w", header.Name, err))
			return err
		}

		// Directories and regular files only: links and devices are skipped, so
		// nothing extracted can point outside the build directory either.
		switch header.Typeflag {
		case tar.TypeDir:
			if err := ostools.MkdirAll(target); err != nil {
				logger.Error(err)
				return ErrWriteDirectoryFailed
			}
		case tar.TypeReg:
			if err := writeFile(target, header, tr); err != nil {
				logger.Error(err)
				return ErrWriteFileFailed
			}
		}
	}
}

// Joins an archive entry name to the build directory, refusing absolute names
// and any name that climbs out of it.
func safeJoin(dir, name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))

	if filepath.IsAbs(cleaned) || filepath.VolumeName(cleaned) != "" ||
		cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", ErrInvalidArchiveEntry
	}

	target := filepath.Join(dir, cleaned)
	rel, err := filepath.Rel(dir, target)

	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrInvalidArchiveEntry
	}

	return target, nil
}

// Writes a regular file, replacing any previous content. Only permission bits
// are kept from the archive: no setuid, setgid or sticky bit.
func writeFile(target string, header *tar.Header, content io.Reader) error {
	if err := ostools.MkdirAll(filepath.Dir(target)); err != nil {
		return err
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)&os.ModePerm)

	if err != nil {
		return err
	}

	if _, err = io.Copy(f, content); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

// Check the file MIME type and size and returns the opened file if valid
func (t *service) tryOpenFile(header *multipart.FileHeader) (file multipart.File, err error) {
	// The size comes from the bytes received, a client cannot lie about it
	if header.Size > t.options.MaxDeploymentArchiveFileSize() {
		return nil, ErrMaxFileSizeExceeded
	}

	if file, err = header.Open(); err != nil {
		return
	}

	defer func() {
		if err == nil {
			return
		}

		// Close the file if an error was thrown
		file.Close()
		file = nil
	}()

	buf := make([]byte, 512)
	n, err := io.ReadFull(file, buf)

	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return
	}

	if http.DetectContentType(buf[:n]) != supportedMimeType {
		err = ErrInvalidFileType
		return
	}

	_, err = file.Seek(0, io.SeekStart)

	return
}
