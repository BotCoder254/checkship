package volumearchive_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type entry struct {
	name     string
	typ      byte
	body     string
	linkname string
	uid      int
	mode     int64
}

// Builds a tar stream like Docker's CopyFromContainer("/v") returns.
func dockerTar(t testing.TB, entries ...entry) io.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, e := range entries {
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}
		hdr := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: mode, Uid: e.uid, Linkname: e.linkname, Size: int64(len(e.body))}
		if e.typ != tar.TypeReg {
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	return &buf
}

func manifest(keys ...string) volumearchive.Manifest {
	m := volumearchive.Manifest{
		CreatedAt:   time.Date(2026, 11, 15, 14, 2, 0, 0, time.UTC),
		AppID:       "app1",
		AppName:     "blog",
		Environment: "production",
		TargetID:    "target1",
		Skipped:     []volumearchive.SkippedMount{{Service: "worker", Kind: "bind", Source: "./cache", Target: "/cache"}},
	}
	for _, k := range keys {
		m.Volumes = append(m.Volumes, volumearchive.ManifestVolume{Key: k, DockerName: "blog-production-app1_" + k})
	}
	return m
}

type restored struct {
	root  *volumearchive.RootInfo
	files map[string]string
	links map[string]string
}

func readAll(t testing.TB, data []byte, include func(string) bool) (volumearchive.Manifest, map[string]restored) {
	t.Helper()
	r, err := volumearchive.Open(bytes.NewReader(data))
	assert.Nil(t, err)
	defer r.Close()

	out := map[string]restored{}
	err = r.Volumes(include, func(key string, root *volumearchive.RootInfo, content io.Reader) error {
		v := restored{root: root, files: map[string]string{}, links: map[string]string{}}
		tr := tar.NewReader(content)
		for {
			hdr, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			switch hdr.Typeflag {
			case tar.TypeReg:
				b, err := io.ReadAll(tr)
				if err != nil {
					return err
				}
				v.files[hdr.Name] = string(b)
			case tar.TypeDir:
				v.files[hdr.Name+"/"] = ""
			case tar.TypeLink, tar.TypeSymlink:
				v.links[hdr.Name] = hdr.Linkname
			}
		}
		out[key] = v
		return nil
	})
	assert.Nil(t, err)
	return r.Manifest(), out
}

func Test_Archive(t *testing.T) {
	t.Run("should round trip volumes keyed by their compose name", func(t *testing.T) {
		var buf bytes.Buffer
		w, err := volumearchive.NewWriter(&buf, manifest("dbdata", "uploads"))
		assert.Nil(t, err)

		stats, err := w.AddVolume("dbdata", dockerTar(t,
			entry{name: "v", typ: tar.TypeDir, uid: 999, mode: 0o700},
			entry{name: "v/PG_VERSION", typ: tar.TypeReg, body: "17"},
			entry{name: "v/base", typ: tar.TypeDir},
			entry{name: "v/base/1", typ: tar.TypeReg, body: "rows"},
			entry{name: "v/base/hard", typ: tar.TypeLink, linkname: "v/base/1"},
			entry{name: "v/latest", typ: tar.TypeSymlink, linkname: "/var/lib/postgresql/data/base/1"},
		), "v")
		assert.Nil(t, err)
		assert.Equal(t, 5, stats.Files)
		assert.Equal(t, int64(6), stats.Size)

		_, err = w.AddVolume("uploads", dockerTar(t,
			entry{name: "./v", typ: tar.TypeDir},
			entry{name: "./v/a.png", typ: tar.TypeReg, body: "png"},
		), "v")
		assert.Nil(t, err)

		summary, err := w.Close()
		assert.Nil(t, err)
		assert.Equal(t, int64(buf.Len()), summary.Size)
		sum := sha256.Sum256(buf.Bytes())
		assert.Equal(t, hex.EncodeToString(sum[:]), summary.SHA256)
		assert.HasLength(t, 2, summary.Volumes)

		m, vols := readAll(t, buf.Bytes(), nil)
		assert.Equal(t, "blog", m.AppName)
		assert.Equal(t, volumearchive.FormatVersion, m.Format)
		assert.HasLength(t, 1, m.Skipped)

		db := vols["dbdata"]
		assert.NotNil(t, db.root)
		assert.Equal(t, 999, db.root.UID)
		assert.Equal(t, int64(0o700), db.root.Mode)
		assert.Equal(t, "17", db.files["PG_VERSION"])
		assert.Equal(t, "rows", db.files["base/1"])
		assert.Equal(t, "", db.files["base/"])
		assert.Equal(t, "base/1", db.links["base/hard"])
		assert.Equal(t, "/var/lib/postgresql/data/base/1", db.links["latest"])
		assert.Equal(t, "png", vols["uploads"].files["a.png"])
	})

	t.Run("should only hand selected volumes to the callback", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := volumearchive.NewWriter(&buf, manifest("a", "b", "c"))
		for _, k := range []string{"a", "b", "c"} {
			_, err := w.AddVolume(k, dockerTar(t, entry{name: "v", typ: tar.TypeDir}, entry{name: "v/f", typ: tar.TypeReg, body: k}), "v")
			assert.Nil(t, err)
		}
		_, err := w.Close()
		assert.Nil(t, err)

		_, vols := readAll(t, buf.Bytes(), func(k string) bool { return k != "b" })
		assert.Equal(t, 2, len(vols))
		assert.Equal(t, "a", vols["a"].files["f"])
		assert.Equal(t, "c", vols["c"].files["f"])
	})

	t.Run("should return the callback error and stop", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := volumearchive.NewWriter(&buf, manifest("a", "b"))
		_, _ = w.AddVolume("a", dockerTar(t, entry{name: "v", typ: tar.TypeDir}, entry{name: "v/f", typ: tar.TypeReg, body: "big"}), "v")
		_, _ = w.AddVolume("b", dockerTar(t, entry{name: "v", typ: tar.TypeDir}), "v")
		_, _ = w.Close()

		r, err := volumearchive.Open(bytes.NewReader(buf.Bytes()))
		assert.Nil(t, err)
		boom := errors.New("container gone")
		calls := 0
		err = r.Volumes(nil, func(string, *volumearchive.RootInfo, io.Reader) error {
			calls++
			return boom
		})
		assert.ErrorIs(t, boom, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("should refuse a volume missing from the manifest or written twice", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := volumearchive.NewWriter(&buf, manifest("a"))
		_, err := w.AddVolume("nope", dockerTar(t), "v")
		assert.ErrorIs(t, volumearchive.ErrUnknownVolume, err)
		_, err = w.AddVolume("a", dockerTar(t, entry{name: "v", typ: tar.TypeDir}), "v")
		assert.Nil(t, err)
		_, err = w.AddVolume("a", dockerTar(t), "v")
		assert.ErrorIs(t, volumearchive.ErrVolumeWritten, err)
	})

	t.Run("should refuse a source entry outside the volume root", func(t *testing.T) {
		var buf bytes.Buffer
		w, _ := volumearchive.NewWriter(&buf, manifest("a"))
		_, err := w.AddVolume("a", dockerTar(t, entry{name: "other/f", typ: tar.TypeReg, body: "x"}), "v")
		assert.ErrorIs(t, volumearchive.ErrInvalidEntry, err)
	})

	t.Run("should refuse invalid volume keys in a manifest", func(t *testing.T) {
		for _, key := range []string{"", "../x", "a/b", "-a", ".a"} {
			_, err := volumearchive.NewWriter(io.Discard, manifest(key))
			assert.ErrorIs(t, volumearchive.ErrInvalidVolumeKey, err, key)
		}
	})
}

// Writes a hand-made archive, to check what Open and Volumes refuse.
func rawArchive(t testing.TB, m *volumearchive.Manifest, entries ...entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if m != nil {
		data, _ := json.Marshal(m)
		_ = tw.WriteHeader(&tar.Header{Name: volumearchive.ManifestName, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(data))})
		_, _ = tw.Write(data)
	}
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: 0o644, Linkname: e.linkname, Size: int64(len(e.body))}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write([]byte(e.body))
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func Test_Open(t *testing.T) {
	t.Run("should refuse something that is not a checkship backup", func(t *testing.T) {
		_, err := volumearchive.Open(bytes.NewReader([]byte("PK\x03\x04 zip file")))
		assert.ErrorIs(t, volumearchive.ErrManifestMissing, err)

		_, err = volumearchive.Open(bytes.NewReader(rawArchive(t, nil, entry{name: "site/index.html", typ: tar.TypeReg, body: "hi"})))
		assert.ErrorIs(t, volumearchive.ErrManifestMissing, err)
	})

	t.Run("should refuse an unknown format version", func(t *testing.T) {
		m := manifest("a")
		m.Format = 99
		_, err := volumearchive.Open(bytes.NewReader(rawArchive(t, &m)))
		assert.ErrorIs(t, volumearchive.ErrUnsupportedFormat, err)
	})

	t.Run("should refuse entries that escape their volume", func(t *testing.T) {
		m := manifest("a", "b")
		m.Format = volumearchive.FormatVersion
		cases := []entry{
			{name: "volumes/a/../../etc/passwd", typ: tar.TypeReg, body: "x"},
			{name: "volumes/a//abs", typ: tar.TypeReg, body: "x"},
			{name: "volumes/a/./f", typ: tar.TypeReg, body: "x"},
			{name: "outside", typ: tar.TypeReg, body: "x"},
			{name: "volumes/zzz/f", typ: tar.TypeReg, body: "x"},
			{name: "volumes/a/link", typ: tar.TypeLink, linkname: "volumes/b/f"},
			{name: "volumes/a/link", typ: tar.TypeLink, linkname: "/etc/shadow"},
			{name: "volumes/a", typ: tar.TypeReg, body: "root is a file"},
		}
		for _, c := range cases {
			r, err := volumearchive.Open(bytes.NewReader(rawArchive(t, &m, c)))
			assert.Nil(t, err)
			err = r.Volumes(nil, func(_ string, _ *volumearchive.RootInfo, content io.Reader) error {
				_, err := io.Copy(io.Discard, content)
				return err
			})
			assert.NotNil(t, err, c.name)
		}
	})
}
