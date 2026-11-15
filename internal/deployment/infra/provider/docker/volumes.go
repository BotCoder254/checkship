package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"sort"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
)

const (
	HelperLabel        = "app.checkship.helper" // Marks the short-lived containers used to copy volumes
	DefaultHelperImage = "busybox:1.37"
	helperMountPath    = "/v"
	helperRootName     = "v" // Name Docker gives to the root entry when copying helperMountPath
	cleanupTimeout     = 2 * time.Minute
)

var anonymousVolumeName = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (d *docker) Volumes(
	ctx context.Context,
	target domain.Target,
	app domain.AppID,
	env domain.Environment,
) ([]domain.VolumeInfo, []volumearchive.SkippedMount, error) {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return nil, nil, err
	}

	defer client.Close()

	return client.appVolumes(ctx, appFilters(app, target, env))
}

func (d *docker) BackupVolumes(
	ctx context.Context,
	target domain.Target,
	app domain.AppID,
	env domain.Environment,
	options domain.BackupVolumesOptions,
	w io.Writer,
) (result domain.BackupVolumesResult, finalErr error) {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return result, err
	}

	defer client.Close()

	criteria := appFilters(app, target, env)
	volumes, skipped, err := client.appVolumes(ctx, criteria)

	if err != nil {
		return result, err
	}

	selected, err := selectVolumes(volumes, options.Volumes)

	if err != nil {
		return result, err
	}

	if len(selected) == 0 {
		return result, domain.ErrNoVolumes
	}

	manifest := options.Manifest
	manifest.Skipped = skipped
	manifest.Volumes = make([]volumearchive.ManifestVolume, len(selected))

	for i, v := range selected {
		manifest.Volumes[i] = volumearchive.ManifestVolume{Key: v.Key, DockerName: v.DockerName, Driver: v.Driver}
	}

	helperImage := helperImageOrDefault(options.HelperImage)

	if err = client.ensureImage(ctx, helperImage); err != nil {
		return result, err
	}

	if options.StopContainers {
		stopped, err := client.stopRunning(ctx, criteria)

		defer func() {
			finalErr = errors.Join(finalErr, client.startAll(stopped))
		}()

		if err != nil {
			return result, err
		}
	}

	archive, err := volumearchive.NewWriter(w, manifest)

	if err != nil {
		return result, err
	}

	for _, v := range selected {
		if err = client.copyVolumeOut(ctx, target, helperImage, v, archive); err != nil {
			return result, err
		}
	}

	summary, err := archive.Close()

	if err != nil {
		return result, err
	}

	return domain.BackupVolumesResult{Manifest: manifest, Summary: summary}, nil
}

func (d *docker) RestoreVolumes(
	ctx context.Context,
	target domain.Target,
	app domain.AppID,
	env domain.Environment,
	options domain.RestoreVolumesOptions,
	r io.Reader,
) (restored []string, finalErr error) {
	archive, err := volumearchive.Open(r)

	if err != nil {
		return nil, err
	}

	defer archive.Close()

	manifest := archive.Manifest()

	for _, key := range options.Volumes {
		if !manifest.HasVolume(key) {
			return nil, domain.ErrVolumeNotFound
		}
	}

	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return nil, err
	}

	defer client.Close()

	criteria := appFilters(app, target, env)
	existing, _, err := client.appVolumes(ctx, criteria)

	if err != nil {
		return nil, err
	}

	byKey := make(map[string]domain.VolumeInfo, len(existing))

	for _, v := range existing {
		byKey[v.Key] = v
	}

	// Explicit keys are restored even if the destination does not declare them yet,
	// otherwise only the volumes it declares.
	include := func(key string) bool {
		if len(options.Volumes) > 0 {
			return slices.Contains(options.Volumes, key)
		}

		_, declared := byKey[key]
		return declared
	}

	if !slices.ContainsFunc(manifest.Volumes, func(v volumearchive.ManifestVolume) bool { return include(v.Key) }) {
		return nil, domain.ErrNoVolumes
	}

	helperImage := helperImageOrDefault(options.HelperImage)

	if err = client.ensureImage(ctx, helperImage); err != nil {
		return nil, err
	}

	stopped, err := client.stopRunning(ctx, criteria)

	defer func() {
		finalErr = errors.Join(finalErr, client.startAll(stopped))
	}()

	if err != nil {
		return nil, err
	}

	err = archive.Volumes(include, func(key string, root *volumearchive.RootInfo, content io.Reader) error {
		v, found := byKey[key]

		if !found {
			created, err := client.createAppVolume(ctx, target, app, env, options.ProjectName, key)

			if err != nil {
				return err
			}

			v = created
		}

		if err := client.copyVolumeIn(ctx, target, helperImage, v, root, content); err != nil {
			return err
		}

		restored = append(restored, key)
		return nil
	})

	return restored, err
}

// Named volumes managed by checkship for the given filters, and the other mounts
// of the matching containers.
func (c *client) appVolumes(ctx context.Context, criteria filters.Args) ([]domain.VolumeInfo, []volumearchive.SkippedMount, error) {
	list, err := c.api.VolumeList(ctx, volume.ListOptions{Filters: criteria})

	if err != nil {
		return nil, nil, err
	}

	byName := make(map[string]*domain.VolumeInfo, len(list.Volumes))
	infos := make([]*domain.VolumeInfo, 0, len(list.Volumes))

	for _, v := range list.Volumes {
		key := v.Labels[api.VolumeLabel]

		if volumearchive.ValidateVolumeKey(key) != nil {
			continue // Not created by compose, nothing to key it on
		}

		info := &domain.VolumeInfo{Key: key, DockerName: v.Name, Driver: v.Driver}
		byName[v.Name] = info
		infos = append(infos, info)
	}

	containers, err := c.api.ContainerList(ctx, container.ListOptions{All: true, Filters: criteria})

	if err != nil {
		return nil, nil, err
	}

	var skipped []volumearchive.SkippedMount
	seen := make(map[string]bool)

	for _, cont := range containers {
		service := cont.Labels[api.ServiceLabel]

		for _, m := range cont.Mounts {
			if m.Type == mount.TypeVolume {
				if info, managed := byName[m.Name]; managed {
					if !slices.Contains(info.Services, service) {
						info.Services = append(info.Services, service)
					}
					continue
				}
			}

			kind := ""

			switch {
			case m.Type == mount.TypeBind:
				kind = "bind"
			case m.Type == mount.TypeVolume && anonymousVolumeName.MatchString(m.Name):
				kind = "anonymous"
			case m.Type == mount.TypeVolume:
				kind = "external"
			default:
				continue // tmpfs, npipe, cluster: nothing on disk to save
			}

			skip := volumearchive.SkippedMount{Service: service, Kind: kind, Source: m.Source, Target: m.Destination}

			if kind == "external" {
				skip.Source = m.Name
			}

			id := skip.Service + "|" + skip.Kind + "|" + skip.Source + "|" + skip.Target

			if !seen[id] {
				seen[id] = true
				skipped = append(skipped, skip)
			}
		}
	}

	result := make([]domain.VolumeInfo, len(infos))

	for i, info := range infos {
		sort.Strings(info.Services)
		result[i] = *info
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	sort.Slice(skipped, func(i, j int) bool {
		if skipped[i].Service != skipped[j].Service {
			return skipped[i].Service < skipped[j].Service
		}
		return skipped[i].Target < skipped[j].Target
	})

	return result, skipped, nil
}

func selectVolumes(volumes []domain.VolumeInfo, keys []string) ([]domain.VolumeInfo, error) {
	if len(keys) == 0 {
		return volumes, nil
	}

	selected := make([]domain.VolumeInfo, 0, len(keys))

	for _, key := range keys {
		idx := slices.IndexFunc(volumes, func(v domain.VolumeInfo) bool { return v.Key == key })

		if idx < 0 {
			return nil, domain.ErrVolumeNotFound
		}

		selected = append(selected, volumes[idx])
	}

	return selected, nil
}

func helperImageOrDefault(img string) string {
	if img == "" {
		return DefaultHelperImage
	}

	return img
}

// Pulls the image when the target does not have it yet.
func (c *client) ensureImage(ctx context.Context, ref string) error {
	if _, _, err := c.api.ImageInspectWithRaw(ctx, ref); err == nil {
		return nil
	} else if !errdefs.IsNotFound(err) {
		return err
	}

	rc, err := c.api.ImagePull(ctx, ref, image.PullOptions{})

	if err != nil {
		return apperr.Wrap(domain.ErrHelperImagePullFailed, err)
	}

	defer rc.Close()

	// The pull happens while the progress stream is read
	if _, err = io.Copy(io.Discard, rc); err != nil {
		return apperr.Wrap(domain.ErrHelperImagePullFailed, err)
	}

	return nil
}

// Stops the running containers matching the filters and returns their IDs.
func (c *client) stopRunning(ctx context.Context, criteria filters.Args) ([]string, error) {
	running := criteria.Clone()
	running.Add("status", "running")

	containers, err := c.api.ContainerList(ctx, container.ListOptions{Filters: running})

	if err != nil {
		return nil, err
	}

	stopped := make([]string, 0, len(containers))

	for _, cont := range containers {
		if err = c.api.ContainerStop(ctx, cont.ID, container.StopOptions{}); err != nil {
			return stopped, err
		}

		stopped = append(stopped, cont.ID)
	}

	return stopped, nil
}

// Starts the given containers again. Runs even when the job context is done, so
// an interrupted copy does not leave the application down.
func (c *client) startAll(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()

	var errs []error

	for _, id := range ids {
		if err := c.api.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
			errs = append(errs, fmt.Errorf("starting container %s again: %w", id, err))
		}
	}

	return errors.Join(errs...)
}

// Creates a container that mounts a volume at helperMountPath. It is only
// started when cmd is given.
func (c *client) createHelper(ctx context.Context, target domain.Target, helperImage, volumeName string, readOnly bool, cmd []string) (string, error) {
	resp, err := c.api.ContainerCreate(ctx,
		&container.Config{
			Image:  helperImage,
			Cmd:    cmd,
			Labels: map[string]string{HelperLabel: "true", TargetLabel: string(target.ID())},
		},
		&container.HostConfig{
			Mounts:      []mount.Mount{{Type: mount.TypeVolume, Source: volumeName, Target: helperMountPath, ReadOnly: readOnly}},
			NetworkMode: "none",
		},
		nil, nil, "checkship-helper-"+id.New[string](),
	)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (c *client) removeHelper(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()

	return c.api.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
}

func (c *client) copyVolumeOut(ctx context.Context, target domain.Target, helperImage string, v domain.VolumeInfo, archive *volumearchive.Writer) (err error) {
	helper, err := c.createHelper(ctx, target, helperImage, v.DockerName, true, nil)

	if err != nil {
		return err
	}

	defer func() { err = errors.Join(err, c.removeHelper(helper)) }()

	content, _, err := c.api.CopyFromContainer(ctx, helper, helperMountPath)

	if err != nil {
		return apperr.Wrap(domain.ErrVolumeCopyFailed, err)
	}

	defer content.Close()

	if _, err = archive.AddVolume(v.Key, content, helperRootName); err != nil {
		return apperr.Wrap(domain.ErrVolumeCopyFailed, err)
	}

	return nil
}

func (c *client) copyVolumeIn(ctx context.Context, target domain.Target, helperImage string, v domain.VolumeInfo, root *volumearchive.RootInfo, content io.Reader) (err error) {
	// Empty the volume first so files removed since the backup do not come back,
	// and give its root the owner and mode it had.
	script := "find " + helperMountPath + " -mindepth 1 -delete"

	if root != nil {
		script += fmt.Sprintf(" && chown %d:%d %s && chmod %o %s", root.UID, root.GID, helperMountPath, root.Mode&0o7777, helperMountPath)
	}

	helper, err := c.createHelper(ctx, target, helperImage, v.DockerName, false, []string{"sh", "-c", script})

	if err != nil {
		return err
	}

	defer func() { err = errors.Join(err, c.removeHelper(helper)) }()

	waitCh, errCh := c.api.ContainerWait(ctx, helper, container.WaitConditionNextExit)

	if err = c.api.ContainerStart(ctx, helper, container.StartOptions{}); err != nil {
		return apperr.Wrap(domain.ErrVolumeCopyFailed, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errCh:
		return apperr.Wrap(domain.ErrVolumeCopyFailed, err)
	case res := <-waitCh:
		if res.StatusCode != 0 {
			return apperr.Wrap(domain.ErrVolumeCopyFailed, fmt.Errorf("emptying volume %s exited with %d", v.DockerName, res.StatusCode))
		}
	}

	if err = c.api.CopyToContainer(ctx, helper, helperMountPath, content, container.CopyToContainerOptions{CopyUIDGID: true}); err != nil {
		return apperr.Wrap(domain.ErrVolumeCopyFailed, err)
	}

	return nil
}

// Creates a volume the way compose would, so the next deployment adopts it
// instead of refusing it.
func (c *client) createAppVolume(ctx context.Context, target domain.Target, app domain.AppID, env domain.Environment, projectName, key string) (domain.VolumeInfo, error) {
	if projectName == "" {
		return domain.VolumeInfo{}, domain.ErrVolumeNotFound
	}

	name := projectName + "_" + key
	created, err := c.api.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: "local",
		Labels: map[string]string{
			api.ProjectLabel: projectName,
			api.VolumeLabel:  key,
			api.VersionLabel: api.ComposeVersion,
			AppLabel:         string(app),
			TargetLabel:      string(target.ID()),
			EnvironmentLabel: string(env),
		},
	})

	if err != nil {
		return domain.VolumeInfo{}, err
	}

	return domain.VolumeInfo{Key: key, DockerName: created.Name, Driver: created.Driver}, nil
}
