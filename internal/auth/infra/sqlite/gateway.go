package sqlite

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/auth/app/get_profile"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

type gateway struct {
	db *sqlite.Database
}

func NewGateway(db *sqlite.Database) *gateway {
	return &gateway{db}
}

func (s *gateway) GetProfile(ctx context.Context, q get_profile.Query) (get_profile.Profile, error) {
	return builder.
		Query[get_profile.Profile](`
			SELECT
				id
				,email
				,registered_at
				,api_key
			FROM users
			WHERE id = ?`, q.ID).
		One(s.db, ctx, profileMapper)
}

func profileMapper(row storage.Scanner) (p get_profile.Profile, err error) {
	err = row.Scan(
		&p.ID,
		&p.Email,
		&p.RegisteredAt,
		&p.APIKey,
	)

	return p, err
}
