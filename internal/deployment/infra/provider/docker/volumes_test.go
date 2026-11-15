package docker_test

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider/docker"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Test_Volumes(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))

	arrange := func(t *testing.T, fake *volumesAPI) docker.Docker {
		cli := &volumesCli{api: fake}
		return docker.New(logger, docker.WithTestConfig(cli, cli, filepath.Join(t.TempDir(), "config")))
	}

	labels := func(env domain.Environment, key string) map[string]string {
		l := map[string]string{
			docker.AppLabel:         "app",
			docker.TargetLabel:      string(target.ID()),
			docker.EnvironmentLabel: string(env),
		}
		if key != "" {
			l[api.VolumeLabel] = key
			l[api.ProjectLabel] = "blog-" + string(env) + "-app"
		}
		return l
	}

	// blog production: db mounts dbdata, app mounts uploads + a bind mount and an anonymous volume.
	seed := func() *volumesAPI {
		fake := newVolumesAPI()
		fake.addVolume("blog-production-app_dbdata", labels(domain.Production, "dbdata"), map[string]string{"PG_VERSION": "17", "base/1": "rows"})
		fake.addVolume("blog-production-app_uploads", labels(domain.Production, "uploads"), map[string]string{"a.png": "png"})
		fake.addVolume("blog-staging-app_dbdata", labels(domain.Staging, "dbdata"), map[string]string{"PG_VERSION": "16", "old": "stale"})
		fake.addContainer("db", "db", labels(domain.Production, ""), "running",
			dockertypes.MountPoint{Type: mount.TypeVolume, Name: "blog-production-app_dbdata", Destination: "/var/lib/postgresql/data"})
		fake.addContainer("app", "app", labels(domain.Production, ""), "running",
			dockertypes.MountPoint{Type: mount.TypeVolume, Name: "blog-production-app_uploads", Destination: "/uploads"},
			dockertypes.MountPoint{Type: mount.TypeBind, Source: "/srv/cache", Destination: "/cache"},
			dockertypes.MountPoint{Type: mount.TypeVolume, Name: strings.Repeat("ab", 32), Destination: "/tmp/data"})
		fake.addContainer("worker", "worker", labels(domain.Production, ""), "exited",
			dockertypes.MountPoint{Type: mount.TypeVolume, Name: "blog-production-app_uploads", Destination: "/uploads"})
		fake.addContainer("stg-db", "db", labels(domain.Staging, ""), "running",
			dockertypes.MountPoint{Type: mount.TypeVolume, Name: "blog-staging-app_dbdata", Destination: "/var/lib/postgresql/data"})
		return fake
	}

	backup := func(t *testing.T, provider docker.Docker, opts domain.BackupVolumesOptions) ([]byte, domain.BackupVolumesResult) {
		var buf bytes.Buffer
		opts.Manifest = volumearchive.Manifest{AppID: "app", AppName: "blog", Environment: "production"}
		result, err := provider.BackupVolumes(context.Background(), target, "app", domain.Production, opts, &buf)
		assert.Nil(t, err)
		return buf.Bytes(), result
	}

	t.Run("should list named volumes with their services and the mounts that are not backed up", func(t *testing.T) {
		provider := arrange(t, seed())

		volumes, skipped, err := provider.Volumes(context.Background(), target, "app", domain.Production)

		assert.Nil(t, err)
		assert.DeepEqual(t, []domain.VolumeInfo{
			{Key: "dbdata", DockerName: "blog-production-app_dbdata", Driver: "local", Services: []string{"db"}},
			{Key: "uploads", DockerName: "blog-production-app_uploads", Driver: "local", Services: []string{"app", "worker"}},
		}, volumes)
		assert.DeepEqual(t, []volumearchive.SkippedMount{
			{Service: "app", Kind: "bind", Source: "/srv/cache", Target: "/cache"},
			{Service: "app", Kind: "anonymous", Target: "/tmp/data"},
		}, skipped)
	})

	t.Run("should stop running containers, copy every volume and start them again", func(t *testing.T) {
		fake := seed()
		provider := arrange(t, fake)

		data, result := backup(t, provider, domain.BackupVolumesOptions{StopContainers: true})

		assert.ArrayEqual(t, []string{"stop app", "stop db", "start app", "start db"}, fake.lifecycle())
		assert.Equal(t, 0, fake.helperCount())
		assert.Equal(t, 2, len(result.Summary.Volumes))
		assert.Equal(t, int64(len(data)), result.Summary.Size)
		assert.HasLength(t, 2, result.Manifest.Skipped)

		r := must.Panic(volumearchive.Open(bytes.NewReader(data)))
		files := map[string]map[string]string{}
		assert.Nil(t, r.Volumes(nil, func(key string, _ *volumearchive.RootInfo, content io.Reader) error {
			files[key] = untar(t, content)
			return nil
		}))
		assert.DeepEqual(t, map[string]string{"PG_VERSION": "17", "base/1": "rows"}, files["dbdata"])
		assert.DeepEqual(t, map[string]string{"a.png": "png"}, files["uploads"])
	})

	t.Run("should keep containers running when asked and copy only the selected volumes", func(t *testing.T) {
		fake := seed()
		provider := arrange(t, fake)

		_, result := backup(t, provider, domain.BackupVolumesOptions{Volumes: []string{"uploads"}})

		assert.HasLength(t, 0, fake.lifecycle())
		assert.HasLength(t, 1, result.Manifest.Volumes)
		assert.Equal(t, "uploads", result.Manifest.Volumes[0].Key)
	})

	t.Run("should pull the helper image when the target does not have it", func(t *testing.T) {
		fake := seed()
		fake.images = map[string]bool{}
		provider := arrange(t, fake)

		backup(t, provider, domain.BackupVolumesOptions{})

		assert.ArrayEqual(t, []string{docker.DefaultHelperImage}, fake.pulled)
	})

	t.Run("should start the containers again and remove the helper when a copy fails", func(t *testing.T) {
		fake := seed()
		fake.copyErr = errors.New("connection reset")
		provider := arrange(t, fake)

		_, err := provider.BackupVolumes(context.Background(), target, "app", domain.Production, domain.BackupVolumesOptions{StopContainers: true}, io.Discard)

		assert.ErrorIs(t, domain.ErrVolumeCopyFailed, err)
		assert.ArrayEqual(t, []string{"stop app", "stop db", "start app", "start db"}, fake.lifecycle())
		assert.Equal(t, 0, fake.helperCount())
	})

	t.Run("should refuse unknown volumes and environments without volumes", func(t *testing.T) {
		provider := arrange(t, seed())

		_, err := provider.BackupVolumes(context.Background(), target, "app", domain.Production, domain.BackupVolumesOptions{Volumes: []string{"nope"}}, io.Discard)
		assert.ErrorIs(t, domain.ErrVolumeNotFound, err)

		_, err = provider.BackupVolumes(context.Background(), target, "other", domain.Production, domain.BackupVolumesOptions{}, io.Discard)
		assert.ErrorIs(t, domain.ErrNoVolumes, err)
	})

	t.Run("should restore into another environment, replacing declared volumes only", func(t *testing.T) {
		fake := seed()
		provider := arrange(t, fake)
		data, _ := backup(t, provider, domain.BackupVolumesOptions{})
		fake.resetLifecycle()

		restored, err := provider.RestoreVolumes(context.Background(), target, "app", domain.Staging, domain.RestoreVolumesOptions{ProjectName: "blog-staging-app"}, bytes.NewReader(data))

		assert.Nil(t, err)
		assert.ArrayEqual(t, []string{"dbdata"}, restored)
		assert.DeepEqual(t, map[string]string{"PG_VERSION": "17", "base/1": "rows"}, fake.files("blog-staging-app_dbdata"))
		assert.ArrayEqual(t, []string{"stop stg-db", "start stg-db"}, fake.lifecycle())
		assert.Equal(t, 0, fake.helperCount())
		assert.False(t, fake.hasVolume("blog-staging-app_uploads"))
	})

	t.Run("should create an explicitly chosen volume the way compose would", func(t *testing.T) {
		fake := seed()
		provider := arrange(t, fake)
		data, _ := backup(t, provider, domain.BackupVolumesOptions{})

		restored, err := provider.RestoreVolumes(context.Background(), target, "app", domain.Staging, domain.RestoreVolumesOptions{
			ProjectName: "blog-staging-app",
			Volumes:     []string{"uploads"},
		}, bytes.NewReader(data))

		assert.Nil(t, err)
		assert.ArrayEqual(t, []string{"uploads"}, restored)
		assert.DeepEqual(t, map[string]string{"a.png": "png"}, fake.files("blog-staging-app_uploads"))
		l := fake.volumeLabels("blog-staging-app_uploads")
		assert.Equal(t, "blog-staging-app", l[api.ProjectLabel])
		assert.Equal(t, "uploads", l[api.VolumeLabel])
		assert.Equal(t, "staging", l[docker.EnvironmentLabel])
		assert.Equal(t, "PG_VERSION=16,old=stale", flatten(fake.files("blog-staging-app_dbdata")))
	})

	t.Run("should refuse a restore of a volume the archive does not have", func(t *testing.T) {
		fake := seed()
		provider := arrange(t, fake)
		data, _ := backup(t, provider, domain.BackupVolumesOptions{Volumes: []string{"uploads"}})

		_, err := provider.RestoreVolumes(context.Background(), target, "app", domain.Staging, domain.RestoreVolumesOptions{Volumes: []string{"dbdata"}}, bytes.NewReader(data))

		assert.ErrorIs(t, domain.ErrVolumeNotFound, err)
		assert.HasLength(t, 0, fake.lifecycle())
	})
}

func untar(t testing.TB, r io.Reader) map[string]string {
	t.Helper()
	files := map[string]string{}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			files[hdr.Name] = string(must.Panic(io.ReadAll(tr)))
		}
	}
}

func flatten(files map[string]string) string {
	parts := make([]string, 0, len(files))
	for k, v := range files {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

type (
	volumesCli struct {
		command.Cli
		api.Service
		api *volumesAPI
	}

	fakeVolume struct {
		labels map[string]string
		files  map[string]string
	}

	fakeAppContainer struct {
		summary dockertypes.Container
	}

	fakeHelper struct {
		volume string
		cmd    []string
	}

	volumesAPI struct {
		client.APIClient

		mu         sync.Mutex
		volumes    map[string]*fakeVolume
		containers []*fakeAppContainer
		helpers    map[string]*fakeHelper
		images     map[string]bool
		pulled     []string
		events     []string
		copyErr    error
		seq        int
	}
)

func (c *volumesCli) Client() client.APIClient { return c.api }

func newVolumesAPI() *volumesAPI {
	return &volumesAPI{
		volumes: map[string]*fakeVolume{},
		helpers: map[string]*fakeHelper{},
		images:  map[string]bool{docker.DefaultHelperImage: true},
	}
}

func (a *volumesAPI) addVolume(name string, labels map[string]string, files map[string]string) {
	a.volumes[name] = &fakeVolume{labels: labels, files: files}
}

func (a *volumesAPI) addContainer(id, service string, labels map[string]string, state string, mounts ...dockertypes.MountPoint) {
	l := map[string]string{api.ServiceLabel: service}
	for k, v := range labels {
		l[k] = v
	}
	a.containers = append(a.containers, &fakeAppContainer{summary: dockertypes.Container{ID: id, Labels: l, State: state, Mounts: mounts}})
}

func (a *volumesAPI) lifecycle() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := append([]string(nil), a.events...)
	sort.SliceStable(out, func(i, j int) bool {
		// stops come before starts, each sorted by container
		si, sj := strings.HasPrefix(out[i], "stop"), strings.HasPrefix(out[j], "stop")
		if si != sj {
			return si
		}
		return out[i] < out[j]
	})
	return out
}

func (a *volumesAPI) resetLifecycle() { a.mu.Lock(); a.events = nil; a.mu.Unlock() }

func (a *volumesAPI) helperCount() int { a.mu.Lock(); defer a.mu.Unlock(); return len(a.helpers) }

func (a *volumesAPI) files(name string) map[string]string { return a.volumes[name].files }

func (a *volumesAPI) hasVolume(name string) bool { _, ok := a.volumes[name]; return ok }

func (a *volumesAPI) volumeLabels(name string) map[string]string { return a.volumes[name].labels }

func (a *volumesAPI) Close() error { return nil }

func matches(labels map[string]string, wanted []string) bool {
	for _, w := range wanted {
		k, v, _ := strings.Cut(w, "=")
		if labels[k] != v {
			return false
		}
	}
	return true
}

func (a *volumesAPI) VolumeList(_ context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	var out volume.ListResponse
	for name, v := range a.volumes {
		if matches(v.labels, options.Filters.Get("label")) {
			out.Volumes = append(out.Volumes, &volume.Volume{Name: name, Driver: "local", Labels: v.labels})
		}
	}
	return out, nil
}

func (a *volumesAPI) VolumeCreate(_ context.Context, options volume.CreateOptions) (volume.Volume, error) {
	a.volumes[options.Name] = &fakeVolume{labels: options.Labels, files: map[string]string{}}
	return volume.Volume{Name: options.Name, Driver: options.Driver}, nil
}

func (a *volumesAPI) ContainerList(_ context.Context, options container.ListOptions) ([]dockertypes.Container, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	status := options.Filters.Get("status")
	var out []dockertypes.Container
	for _, c := range a.containers {
		if !matches(c.summary.Labels, options.Filters.Get("label")) {
			continue
		}
		if len(status) > 0 && c.summary.State != status[0] {
			continue
		}
		out = append(out, c.summary)
	}
	return out, nil
}

func (a *volumesAPI) ImageInspectWithRaw(_ context.Context, ref string) (dockertypes.ImageInspect, []byte, error) {
	if a.images[ref] {
		return dockertypes.ImageInspect{ID: ref}, nil, nil
	}
	return dockertypes.ImageInspect{}, nil, errdefs.NotFound(errors.New("no such image"))
}

func (a *volumesAPI) ImagePull(_ context.Context, ref string, _ image.PullOptions) (io.ReadCloser, error) {
	a.pulled = append(a.pulled, ref)
	a.images[ref] = true
	return io.NopCloser(strings.NewReader(`{"status":"done"}`)), nil
}

func (a *volumesAPI) appContainer(id string) *fakeAppContainer {
	for _, c := range a.containers {
		if c.summary.ID == id {
			return c
		}
	}
	return nil
}

func (a *volumesAPI) ContainerStop(_ context.Context, id string, _ container.StopOptions) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appContainer(id).summary.State = "exited"
	a.events = append(a.events, "stop "+id)
	return nil
}

func (a *volumesAPI) ContainerStart(_ context.Context, id string, _ container.StartOptions) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if h, ok := a.helpers[id]; ok {
		// The only command a helper runs empties its volume
		if len(h.cmd) > 0 {
			a.volumes[h.volume].files = map[string]string{}
		}
		return nil
	}
	a.appContainer(id).summary.State = "running"
	a.events = append(a.events, "start "+id)
	return nil
}

func (a *volumesAPI) ContainerWait(_ context.Context, _ string, _ container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	res := make(chan container.WaitResponse, 1)
	res <- container.WaitResponse{StatusCode: 0}
	return res, make(chan error)
}

func (a *volumesAPI) ContainerCreate(_ context.Context, config *container.Config, host *container.HostConfig, _ *network.NetworkingConfig, _ *ocispec.Platform, name string) (container.CreateResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.images[config.Image] {
		return container.CreateResponse{}, errdefs.NotFound(errors.New("no such image"))
	}
	if config.Labels[docker.HelperLabel] != "true" || host.NetworkMode != "none" {
		return container.CreateResponse{}, errors.New("helpers must be labelled and offline")
	}
	a.seq++
	id := name
	a.helpers[id] = &fakeHelper{volume: host.Mounts[0].Source, cmd: config.Cmd}
	return container.CreateResponse{ID: id}, nil
}

func (a *volumesAPI) ContainerRemove(_ context.Context, id string, _ container.RemoveOptions) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.helpers, id)
	return nil
}

func (a *volumesAPI) CopyFromContainer(_ context.Context, id, path string) (io.ReadCloser, container.PathStat, error) {
	if a.copyErr != nil {
		return nil, container.PathStat{}, a.copyErr
	}
	h := a.helpers[id]
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "v", Typeflag: tar.TypeDir, Mode: 0o755})
	names := make([]string, 0)
	for n := range a.volumes[h.volume].files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		body := a.volumes[h.volume].files[n]
		_ = tw.WriteHeader(&tar.Header{Name: "v/" + n, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))})
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	return io.NopCloser(&buf), container.PathStat{Name: "v"}, nil
}

func (a *volumesAPI) CopyToContainer(_ context.Context, id, path string, content io.Reader, _ container.CopyToContainerOptions) error {
	h := a.helpers[id]
	files := map[string]string{}
	tr := tar.NewReader(content)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeReg {
			files[hdr.Name] = string(must.Panic(io.ReadAll(tr)))
		}
	}
	for k, v := range files {
		a.volumes[h.volume].files[k] = v
	}
	return nil
}
