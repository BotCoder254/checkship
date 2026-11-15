package get_backup_archive

import (
	"context"
	"io"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

type (
	// Opens the archive of a finished backup for download. The caller closes it.
	Query struct {
		bus.Query[Archive]

		ID string `json:"-"`
	}

	Archive struct {
		Content  io.ReadCloser
		Size     int64
		Filename string // <app>-<environment>-<date>.tar.gz
	}
)

func (Query) Name_() string { return "deployment.query.get_backup_archive" }

func Handler(reader domain.BackupsReader, files domain.BackupFiles) bus.RequestHandler[Archive, Query] {
	return func(ctx context.Context, q Query) (Archive, error) {
		backup, err := reader.GetByID(ctx, domain.BackupID(q.ID))

		if err != nil {
			return Archive{}, err
		}

		if _, err = backup.Restorable(); err != nil {
			return Archive{}, err
		}

		content, size, err := files.Open(ctx, backup)

		if err != nil {
			return Archive{}, err
		}

		return Archive{
			Content:  content,
			Size:     size,
			Filename: filename(backup),
		}, nil
	}
}

// A name safe in a Content-Disposition header: app names are already
// restricted to lower case letters, digits and dashes.
func filename(b domain.Backup) string {
	name := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, string(b.AppName()))

	return name + "-" + string(b.Environment()) + "-" + b.Requested().At().UTC().Format("2006-01-02-1504") + ".tar.gz"
}
