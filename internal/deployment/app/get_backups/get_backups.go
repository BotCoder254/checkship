package get_backups

import (
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

type (
	// Backups, newest first, of one application or of every one (including
	// applications that were deleted since).
	Query struct {
		bus.Query[[]Backup]

		AppID monad.Maybe[string] `form:"app_id"`
	}

	// One backup with the restores made from it, newest first.
	ByIDQuery struct {
		bus.Query[BackupDetail]

		ID string `json:"-"`
	}

	Backup struct {
		ID             string                           `json:"id"`
		App            AppSummary                       `json:"app"`
		Environment    string                           `json:"environment"`
		Target         TargetSummary                    `json:"target"`
		Trigger        string                           `json:"trigger"`
		Status         string                           `json:"status"`
		StopContainers bool                             `json:"stop_containers"`
		Volumes        []string                         `json:"volumes"` // Asked for, all if empty
		Result         monad.Maybe[domain.BackupResult] `json:"result"`
		Local          bool                             `json:"local"`
		Stored         bool                             `json:"stored"`
		StoreError     monad.Maybe[string]              `json:"store_error"`
		ErrorCode      monad.Maybe[string]              `json:"error_code"`
		RequestedAt    time.Time                        `json:"requested_at"`
		RequestedBy    app.UserSummary                  `json:"requested_by"`
		FinishedAt     monad.Maybe[time.Time]           `json:"finished_at"`
	}

	BackupDetail struct {
		Backup
		Restores []Restore `json:"restores"`
	}

	AppSummary struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Deleted bool   `json:"deleted"` // The backup outlived its application
	}

	TargetSummary struct {
		ID   string              `json:"id"`
		Name monad.Maybe[string] `json:"name"` // Not set when the target was deleted
	}

	Restore struct {
		ID          string                 `json:"id"`
		App         AppSummary             `json:"app"`
		Environment string                 `json:"environment"`
		Status      string                 `json:"status"`
		Volumes     []string               `json:"volumes"`
		Restored    []string               `json:"restored"`
		SnapshotID  monad.Maybe[string]    `json:"snapshot_id"`
		ErrorCode   monad.Maybe[string]    `json:"error_code"`
		RequestedAt time.Time              `json:"requested_at"`
		RequestedBy app.UserSummary        `json:"requested_by"`
		FinishedAt  monad.Maybe[time.Time] `json:"finished_at"`
	}
)

func (Query) Name_() string     { return "deployment.query.get_backups" }
func (ByIDQuery) Name_() string { return "deployment.query.get_backup" }
