// Package volumearchive reads and writes checkship volume backups.
//
// A backup is one gzipped tar. Its first entry is manifest.json, then the
// content of each volume under volumes/<key>/, one volume after the other.
// Volumes are keyed by their name in compose.yml (not the Docker name), so a
// backup taken in one environment can be restored into another one.
package volumearchive

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/pkg/apperr"
)

const (
	FormatVersion = 1
	ManifestName  = "manifest.json"
	volumesDir    = "volumes/"
	maxManifest   = 1 << 20 // A manifest is a few kilobytes, refuse anything absurd
)

var (
	ErrManifestMissing   = apperr.New("manifest_missing")
	ErrUnsupportedFormat = apperr.New("unsupported_archive_format")
	ErrInvalidEntry      = apperr.New("invalid_archive_entry")
	ErrInvalidVolumeKey  = apperr.New("invalid_volume_key")
	ErrUnknownVolume     = apperr.New("unknown_volume")
	ErrVolumeWritten     = apperr.New("volume_already_written")

	// Compose volume names: a letter or digit first, then letters, digits, '_', '.', '-'.
	volumeKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
)

type (
	Manifest struct {
		Format      int              `json:"format"`
		Checkship   string           `json:"checkship,omitempty"`
		CreatedAt   time.Time        `json:"created_at"`
		AppID       string           `json:"app_id"`
		AppName     string           `json:"app_name"`
		Environment string           `json:"environment"`
		TargetID    string           `json:"target_id"`
		Volumes     []ManifestVolume `json:"volumes"`
		Skipped     []SkippedMount   `json:"skipped,omitempty"`
	}

	ManifestVolume struct {
		Key        string `json:"key"`
		DockerName string `json:"docker_name,omitempty"`
		Driver     string `json:"driver,omitempty"`
	}

	// A mount found in the compose project that is not part of the backup.
	SkippedMount struct {
		Service string `json:"service"`
		Kind    string `json:"kind"` // bind, anonymous or external
		Source  string `json:"source,omitempty"`
		Target  string `json:"target"`
	}

	VolumeStats struct {
		Key   string `json:"key"`
		Size  int64  `json:"size"`  // Bytes of regular file content
		Files int    `json:"files"` // Number of entries, directories included
	}

	Summary struct {
		Size    int64         // Bytes of the compressed archive
		SHA256  string        // Hex digest of the compressed archive
		Volumes []VolumeStats // In the order they were written
	}

	// Ownership and mode of a volume's root directory, so a restore can set
	// them back on a newly created volume.
	RootInfo struct {
		UID  int
		GID  int
		Mode int64
	}
)

func ValidateVolumeKey(key string) error {
	if !volumeKeyRegex.MatchString(key) {
		return ErrInvalidVolumeKey
	}

	return nil
}

func (m Manifest) HasVolume(key string) bool {
	for _, v := range m.Volumes {
		if v.Key == key {
			return true
		}
	}

	return false
}

func (m Manifest) validate() error {
	if m.Format != FormatVersion {
		return ErrUnsupportedFormat
	}

	seen := make(map[string]bool, len(m.Volumes))

	for _, v := range m.Volumes {
		if err := ValidateVolumeKey(v.Key); err != nil {
			return err
		}

		if seen[v.Key] {
			return ErrInvalidVolumeKey
		}

		seen[v.Key] = true
	}

	return nil
}

// Writer builds an archive. Call AddVolume once per manifest volume, then Close.
type Writer struct {
	counter  *countingWriter
	hash     hash.Hash
	gz       *gzip.Writer
	tw       *tar.Writer
	manifest Manifest
	written  map[string]bool
	stats    []VolumeStats
}

// Starts an archive on w and writes the manifest as its first entry.
func NewWriter(w io.Writer, manifest Manifest) (*Writer, error) {
	if manifest.Format == 0 {
		manifest.Format = FormatVersion
	}

	if err := manifest.validate(); err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")

	if err != nil {
		return nil, err
	}

	h := sha256.New()
	counter := &countingWriter{w: io.MultiWriter(w, h)}
	gz := gzip.NewWriter(counter)
	tw := tar.NewWriter(gz)

	if err = tw.WriteHeader(&tar.Header{
		Name:     ManifestName,
		Mode:     0o644,
		Size:     int64(len(data)),
		ModTime:  manifest.CreatedAt,
		Typeflag: tar.TypeReg,
		Format:   tar.FormatPAX,
	}); err != nil {
		return nil, err
	}

	if _, err = tw.Write(data); err != nil {
		return nil, err
	}

	return &Writer{
		counter:  counter,
		hash:     h,
		gz:       gz,
		tw:       tw,
		manifest: manifest,
		written:  make(map[string]bool, len(manifest.Volumes)),
	}, nil
}

// Copies a tar stream of one volume into the archive. root is the name the
// source stream gives to the volume's root directory (Docker names it after
// the last segment of the copied path, "v" for "/v"), "" if entries are
// already relative to the volume.
func (w *Writer) AddVolume(key string, src io.Reader, root string) (VolumeStats, error) {
	stats := VolumeStats{Key: key}

	if !w.manifest.HasVolume(key) {
		return stats, ErrUnknownVolume
	}

	if w.written[key] {
		return stats, ErrVolumeWritten
	}

	w.written[key] = true
	prefix := volumesDir + key + "/"
	tr := tar.NewReader(src)

	for {
		hdr, err := tr.Next()

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return stats, fmt.Errorf("reading volume %s: %w", key, err)
		}

		rel, err := relativeTo(hdr.Name, root)

		if err != nil {
			return stats, err
		}

		out := *hdr
		out.Name = prefix + rel
		out.Format = tar.FormatPAX

		if rel == "" && hdr.Typeflag != tar.TypeDir {
			return stats, ErrInvalidEntry // The root of a volume is always a directory
		}

		if hdr.Typeflag == tar.TypeDir && !strings.HasSuffix(out.Name, "/") {
			out.Name += "/"
		}

		if hdr.Typeflag == tar.TypeLink {
			target, err := relativeTo(hdr.Linkname, root)

			if err != nil || target == "" {
				return stats, ErrInvalidEntry
			}

			out.Linkname = prefix + target
		}

		if err = w.tw.WriteHeader(&out); err != nil {
			return stats, err
		}

		if hdr.Typeflag == tar.TypeReg {
			n, err := io.Copy(w.tw, tr)

			if err != nil {
				return stats, fmt.Errorf("copying %s in volume %s: %w", rel, key, err)
			}

			stats.Size += n
		}

		if rel != "" {
			stats.Files++ // The root is not a file of the volume
		}
	}

	w.stats = append(w.stats, stats)

	return stats, nil
}

// Flushes the archive. It does not close the underlying writer.
func (w *Writer) Close() (Summary, error) {
	if err := w.tw.Close(); err != nil {
		return Summary{}, err
	}

	if err := w.gz.Close(); err != nil {
		return Summary{}, err
	}

	return Summary{
		Size:    w.counter.n,
		SHA256:  hex.EncodeToString(w.hash.Sum(nil)),
		Volumes: w.stats,
	}, nil
}

// Reader reads an archive written by Writer.
type Reader struct {
	gz       *gzip.Reader
	tr       *tar.Reader
	manifest Manifest
}

// Opens an archive and reads its manifest, which must be the first entry.
func Open(r io.Reader) (*Reader, error) {
	gz, err := gzip.NewReader(r)

	if err != nil {
		return nil, ErrManifestMissing
	}

	tr := tar.NewReader(gz)
	hdr, err := tr.Next()

	if err != nil || hdr.Name != ManifestName || hdr.Typeflag != tar.TypeReg || hdr.Size > maxManifest {
		gz.Close()
		return nil, ErrManifestMissing
	}

	var manifest Manifest

	if err = json.NewDecoder(io.LimitReader(tr, maxManifest)).Decode(&manifest); err != nil {
		gz.Close()
		return nil, ErrManifestMissing
	}

	if err = manifest.validate(); err != nil {
		gz.Close()
		return nil, err
	}

	return &Reader{gz: gz, tr: tr, manifest: manifest}, nil
}

func (r *Reader) Manifest() Manifest { return r.manifest }

func (r *Reader) Close() error { return r.gz.Close() }

// Streams each volume to fn as a tar whose entries are relative to the volume
// root (the root entry itself is not part of the stream, its ownership is given
// in root, nil if the archive has none). include decides which volumes are
// handed to fn; the others are skipped. fn must read content until EOF.
func (r *Reader) Volumes(include func(key string) bool, fn func(key string, root *RootInfo, content io.Reader) error) error {
	var current *volumeStream

	finish := func(cause error) error {
		if current == nil {
			return cause
		}

		err := current.finish(cause)
		current = nil

		if cause != nil {
			return cause
		}

		return err
	}

	var currentKey string

	for {
		hdr, err := r.tr.Next()

		if errors.Is(err, io.EOF) {
			return finish(nil)
		}

		if err != nil {
			return finish(err)
		}

		key, rel, err := splitEntry(hdr.Name)

		if err != nil {
			return finish(err)
		}

		if !r.manifest.HasVolume(key) {
			return finish(ErrUnknownVolume)
		}

		if key != currentKey {
			if err = finish(nil); err != nil {
				return err
			}

			currentKey = key

			if include == nil || include(key) {
				var root *RootInfo

				if rel == "" && hdr.Typeflag == tar.TypeDir {
					root = &RootInfo{UID: hdr.Uid, GID: hdr.Gid, Mode: hdr.Mode}
				}

				current = startVolumeStream(key, root, fn)
			}
		}

		if rel == "" {
			if hdr.Typeflag != tar.TypeDir {
				return finish(ErrInvalidEntry)
			}

			continue // The root is described by RootInfo
		}

		if hdr.Typeflag == tar.TypeLink {
			linkKey, linkRel, err := splitEntry(hdr.Linkname)

			if err != nil || linkKey != key || linkRel == "" {
				return finish(ErrInvalidEntry)
			}

			hdr.Linkname = linkRel
		}

		if current == nil {
			continue // Volume not selected, tar.Reader skips its content
		}

		out := *hdr
		out.Name = rel

		if err = current.write(&out, r.tr); err != nil {
			return finish(err)
		}
	}
}

type volumeStream struct {
	pw   *io.PipeWriter
	tw   *tar.Writer
	done chan error
}

func startVolumeStream(key string, root *RootInfo, fn func(string, *RootInfo, io.Reader) error) *volumeStream {
	pr, pw := io.Pipe()
	s := &volumeStream{pw: pw, tw: tar.NewWriter(pw), done: make(chan error, 1)}

	go func() {
		err := fn(key, root, pr)
		// Unblock the writer if fn stopped reading early
		pr.CloseWithError(errors.Join(err, io.ErrClosedPipe))
		s.done <- err
	}()

	return s
}

func (s *volumeStream) write(hdr *tar.Header, content io.Reader) error {
	if err := s.tw.WriteHeader(hdr); err != nil {
		return s.finish(err)
	}

	if hdr.Typeflag == tar.TypeReg {
		if _, err := io.Copy(s.tw, content); err != nil {
			return s.finish(err)
		}
	}

	return nil
}

func (s *volumeStream) finish(cause error) error {
	if s.done == nil {
		return cause
	}

	if cause == nil {
		cause = s.tw.Close()
	}

	s.pw.CloseWithError(cause)
	err := <-s.done
	s.done = nil

	if err != nil {
		return err
	}

	return cause
}

// Splits "volumes/<key>/<rel>" and refuses anything that could land outside
// the volume once extracted.
func splitEntry(name string) (key, rel string, err error) {
	rest, ok := strings.CutPrefix(name, volumesDir)

	if !ok {
		return "", "", ErrInvalidEntry
	}

	key, rel, _ = strings.Cut(rest, "/")

	if ValidateVolumeKey(key) != nil {
		return "", "", ErrInvalidEntry
	}

	rel = strings.TrimSuffix(rel, "/")

	if rel == "" {
		return key, "", nil
	}

	if path.IsAbs(rel) || path.Clean(rel) != rel {
		return "", "", ErrInvalidEntry
	}

	for _, part := range strings.Split(rel, "/") {
		if part == ".." || part == "." {
			return "", "", ErrInvalidEntry
		}
	}

	return key, rel, nil
}

// Name of an entry relative to root, "" for root itself.
func relativeTo(name, root string) (string, error) {
	name = strings.TrimPrefix(name, "./")
	clean := path.Clean("/" + name)[1:]

	if clean == "." {
		clean = ""
	}

	if root == "" {
		return clean, nil
	}

	if clean == root {
		return "", nil
	}

	rel, ok := strings.CutPrefix(clean, root+"/")

	if !ok {
		return "", ErrInvalidEntry
	}

	return rel, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
