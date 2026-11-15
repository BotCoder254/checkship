package archive_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/archive"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

type entry struct {
	name string
	typ  byte
	body string
	mode int64
	link string
}

func targz(t testing.TB, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}
		hdr := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: mode, Linkname: e.link}
		if e.typ == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typ == tar.TypeReg {
			_, _ = tw.Write([]byte(e.body))
		}
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

// A multipart file header, the way gin hands an uploaded file to the source.
func upload(t testing.TB, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("archive", "app.tar.gz")
	_, _ = part.Write(content)
	_ = w.Close()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	return req.MultipartForm.File["archive"][0]
}

type options int64

func (o options) MaxDeploymentArchiveFileSize() int64 { return int64(o) }

type discardLogger struct{ io.Writer }

func (discardLogger) Close() error         { return nil }
func (discardLogger) Stepf(string, ...any) {}
func (discardLogger) Warnf(string, ...any) {}
func (discardLogger) Infof(string, ...any) {}
func (discardLogger) Error(error)          {}

func Test_Archive(t *testing.T) {
	ctx := context.Background()
	app := fixture.App()

	// Prepares the archive like a deployment request, then extracts it in a fresh build directory.
	extract := func(t *testing.T, content []byte) (string, error) {
		t.Helper()
		src := archive.New(options(1 << 20))
		data, err := src.Prepare(ctx, app, upload(t, content))
		if err != nil {
			return "", err
		}
		t.Cleanup(func() { os.Remove(string(data.(archive.Data))) })

		root := t.TempDir()
		build := filepath.Join(root, "apps", "app", "production")
		assert.Nil(t, os.MkdirAll(build, 0o755))

		depl := fixture.Deployment(fixture.FromApp(app), fixture.WithSourceData(data))
		return root, src.Fetch(ctx, domain.NewDeploymentContext(build, discardLogger{io.Discard}), depl)
	}

	t.Run("should extract a regular archive", func(t *testing.T) {
		root, err := extract(t, targz(t,
			entry{name: "compose.yml", typ: tar.TypeReg, body: "services: {}"},
			entry{name: "src/", typ: tar.TypeDir, mode: 0o755},
			entry{name: "src/main.go", typ: tar.TypeReg, body: "package main", mode: 0o4755},
		))

		assert.Nil(t, err)
		build := filepath.Join(root, "apps", "app", "production")
		content, _ := os.ReadFile(filepath.Join(build, "compose.yml"))
		assert.Equal(t, "services: {}", string(content))
		info, err := os.Stat(filepath.Join(build, "src", "main.go"))
		assert.Nil(t, err)
		assert.Equal(t, os.FileMode(0), info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky))
	})

	t.Run("should refuse entries that escape the build directory", func(t *testing.T) {
		for _, name := range []string{"../../escaped.txt", "src/../../../escaped.txt", "/tmp/escaped.txt", ".."} {
			root, err := extract(t, targz(t,
				entry{name: "compose.yml", typ: tar.TypeReg, body: "services: {}"},
				entry{name: name, typ: tar.TypeReg, body: "pwned"},
			))

			assert.ErrorIs(t, archive.ErrInvalidArchiveEntry, err, name)
			_, statErr := os.Stat(filepath.Join(root, "apps", "escaped.txt"))
			assert.True(t, os.IsNotExist(statErr), name)
			_, statErr = os.Stat(filepath.Join(root, "escaped.txt"))
			assert.True(t, os.IsNotExist(statErr), name)
		}
	})

	t.Run("should refuse a directory entry that escapes the build directory", func(t *testing.T) {
		root, err := extract(t, targz(t, entry{name: "../outside/", typ: tar.TypeDir, mode: 0o755}))

		assert.ErrorIs(t, archive.ErrInvalidArchiveEntry, err)
		_, statErr := os.Stat(filepath.Join(root, "apps", "app", "outside"))
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should truncate a file written twice", func(t *testing.T) {
		root, err := extract(t, targz(t,
			entry{name: "a.txt", typ: tar.TypeReg, body: "a long first version"},
			entry{name: "a.txt", typ: tar.TypeReg, body: "short"},
		))

		assert.Nil(t, err)
		content, _ := os.ReadFile(filepath.Join(root, "apps", "app", "production", "a.txt"))
		assert.Equal(t, "short", string(content))
	})

	t.Run("should refuse an archive over the size limit", func(t *testing.T) {
		src := archive.New(options(10))

		_, err := src.Prepare(ctx, app, upload(t, targz(t, entry{name: "compose.yml", typ: tar.TypeReg, body: "services: {}"})))

		assert.ValidationError(t, validate.FieldErrors{"archive": archive.ErrMaxFileSizeExceeded}, err)
	})

	t.Run("should refuse a file that is not a gzipped archive", func(t *testing.T) {
		src := archive.New(options(1 << 20))

		_, err := src.Prepare(ctx, app, upload(t, []byte("PK\x03\x04 this is a zip file")))

		assert.ValidationError(t, validate.FieldErrors{"archive": archive.ErrInvalidFileType}, err)
	})
}
