package request_backup

import (
	"context"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

// Back up the volumes of an application environment now.
type Command struct {
	bus.Command[string]

	AppID          string   `json:"-"`
	Environment    string   `json:"environment"`
	Volumes        []string `json:"volumes"`         // Every volume if empty
	StopContainers bool     `json:"stop_containers"` // The dashboard sends true unless "Keep running" is on
}

func (Command) Name_() string { return "deployment.command.request_backup" }

func Handler(
	apps domain.AppsReader,
	writer domain.BackupsWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var env domain.Environment

		if err := validate.Struct(validate.Of{
			"environment": validate.Value(cmd.Environment, &env, domain.EnvironmentFrom),
		}); err != nil {
			return "", err
		}

		app, err := apps.GetByID(ctx, domain.AppID(cmd.AppID))

		if err != nil {
			return "", err
		}

		backup, err := domain.NewBackup(app, env, domain.BackupTriggerManual, cmd.Volumes, cmd.StopContainers, auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return "", err
		}

		if err = writer.Write(ctx, &backup); err != nil {
			return "", err
		}

		return string(backup.ID()), nil
	}
}
