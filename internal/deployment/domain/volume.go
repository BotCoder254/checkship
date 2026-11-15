package domain

import (
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

var (
	ErrNoVolumes             = apperr.New("no_volumes")
	ErrVolumeNotFound        = apperr.New("volume_not_found")
	ErrHelperImagePullFailed = apperr.New("helper_image_pull_failed")
	ErrVolumeCopyFailed      = apperr.New("volume_copy_failed")
)

type (
	// A named volume declared by an application environment, as it exists on a target.
	VolumeInfo struct {
		Key        string   // Name in compose.yml
		DockerName string   // Name on the Docker host
		Driver     string   // Volume driver, "local" most of the time
		Services   []string // Services mounting it, sorted
	}

	// What the provider needs to copy volumes out of a target.
	BackupVolumesOptions struct {
		Manifest       volumearchive.Manifest // Written in the archive, volumes and skipped mounts are filled by the provider
		Volumes        []string               // Keys to copy, every volume if empty
		StopContainers bool                   // Stop the environment's running containers during the copy
		HelperImage    string                 // Image used to mount each volume
	}

	BackupVolumesResult struct {
		Manifest volumearchive.Manifest
		Summary  volumearchive.Summary
	}

	// What the provider needs to put volumes back on a target. The containers using
	// them are always stopped while their files are replaced.
	RestoreVolumesOptions struct {
		ProjectName string   // Compose project of the destination, used to name volumes created by the restore
		Volumes     []string // Keys to restore, every volume the destination declares if empty
		HelperImage string
	}
)

type (
	// What the built-in storage needs to run on a target.
	BuiltinStorageSpec struct {
		Image     string
		AccessKey string
		SecretKey string
	}

	// One way to reach the built-in storage. Candidates are tried in order.
	StorageEndpoint struct {
		Address string // host[:port]
		UseSSL  bool
	}
)
