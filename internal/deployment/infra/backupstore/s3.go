package backupstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Archives are uploaded in parts of this size, so memory stays flat whatever the archive size.
const partSize = 64 << 20

// A bucket in an S3-compatible storage, through minio-go. minio-go sends no
// checksum headers a server did not ask for, unlike aws-sdk-go-v2 since v1.73,
// which several self-hosted servers reject.
type s3Remote struct {
	client *minio.Client
	bucket string
	prefix string
}

func newS3(access domain.BackupStorageAccess) (*s3Remote, error) {
	lookup := minio.BucketLookupDNS

	if access.PathStyle {
		lookup = minio.BucketLookupPath
	}

	client, err := minio.New(access.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(access.AccessKey, string(access.SecretKey), ""),
		Secure:       access.UseSSL,
		Region:       access.Region,
		BucketLookup: lookup,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 2 * time.Minute,
			IdleConnTimeout:       90 * time.Second,
		},
	})

	if err != nil {
		return nil, err
	}

	prefix := strings.Trim(access.Prefix, "/")

	if prefix != "" {
		prefix += "/"
	}

	return &s3Remote{client: client, bucket: access.Bucket, prefix: prefix}, nil
}

func (r *s3Remote) Put(ctx context.Context, key string, content io.Reader, size int64) error {
	_, err := r.client.PutObject(ctx, r.bucket, r.prefix+key, content, size, minio.PutObjectOptions{
		ContentType: "application/gzip",
		PartSize:    partSize,
	})

	return err
}

func (r *s3Remote) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	obj, err := r.client.GetObject(ctx, r.bucket, r.prefix+key, minio.GetObjectOptions{})

	if err != nil {
		return nil, 0, err
	}

	info, err := obj.Stat()

	if err != nil {
		obj.Close()
		return nil, 0, err
	}

	return obj, info.Size, nil
}

func (r *s3Remote) Remove(ctx context.Context, key string) error {
	return r.client.RemoveObject(ctx, r.bucket, r.prefix+key, minio.RemoveObjectOptions{})
}

// Resolves the storage from the settings saved in the database, read on every
// call so a change on the Storage page applies to the next backup.
func Resolver(reader domain.BackupStorageReader) RemoteResolver {
	return func(ctx context.Context) (Remote, bool, error) {
		settings, err := reader.Get(ctx)

		if err != nil {
			return nil, true, err
		}

		access, err := settings.Remote()

		if err != nil {
			return nil, true, err
		}

		a, isSet := access.TryGet()

		if !isSet {
			return nil, true, nil
		}

		remote, err := newS3(a)

		if err != nil {
			return nil, true, err
		}

		return remote, settings.KeepLocal(), nil
	}
}

type checker struct{}

// Checks storages by creating the bucket if needed, then writing, reading and
// deleting a small object.
func NewChecker() domain.BackupStorageChecker { return checker{} }

func (checker) Check(ctx context.Context, access domain.BackupStorageAccess) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	remote, err := newS3(access)

	if err != nil {
		return err
	}

	exists, err := remote.client.BucketExists(ctx, access.Bucket)

	if err != nil {
		return err
	}

	if !exists {
		if err = remote.client.MakeBucket(ctx, access.Bucket, minio.MakeBucketOptions{Region: access.Region}); err != nil {
			return fmt.Errorf("creating bucket %s: %w", access.Bucket, err)
		}
	}

	key := ".checkship-check-" + id.New[string]()
	payload := []byte("checkship storage check")

	if err = remote.Put(ctx, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
		return err
	}

	rc, _, err := remote.Get(ctx, key)

	if err != nil {
		return err
	}

	got, err := io.ReadAll(rc)
	rc.Close()

	if err != nil {
		return err
	}

	if !bytes.Equal(got, payload) {
		return errors.New("the object read back differs from the one written")
	}

	return remote.Remove(ctx, key)
}

type mover struct{}

// Copies archives between two storages, keeping their keys.
func NewMover() mover { return mover{} }

// Copies every object under the source prefix, then checks each one arrived
// with the same size. Returns how many were copied.
func (mover) Move(ctx context.Context, from, to domain.BackupStorageAccess) (int, error) {
	src, err := newS3(from)

	if err != nil {
		return 0, err
	}

	dst, err := newS3(to)

	if err != nil {
		return 0, err
	}

	count := 0

	for obj := range src.client.ListObjects(ctx, from.Bucket, minio.ListObjectsOptions{Prefix: src.prefix, Recursive: true}) {
		if obj.Err != nil {
			return count, obj.Err
		}

		key := strings.TrimPrefix(obj.Key, src.prefix)

		if strings.HasPrefix(key, ".checkship-check-") {
			continue
		}

		rc, size, err := src.Get(ctx, key)

		if err != nil {
			return count, err
		}

		err = dst.Put(ctx, key, rc, size)
		rc.Close()

		if err != nil {
			return count, err
		}

		info, err := dst.client.StatObject(ctx, to.Bucket, dst.prefix+key, minio.StatObjectOptions{})

		if err != nil {
			return count, err
		}

		if info.Size != size {
			return count, fmt.Errorf("%s arrived with %d bytes instead of %d", key, info.Size, size)
		}

		count++
	}

	return count, nil
}
