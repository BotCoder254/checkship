package sqlite

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

type cloudflareStore struct {
	db *sqlite.Database
}

// Stores Connect Cloudflare's connections, one per target.
func NewCloudflareConnectionsStore(db *sqlite.Database) domain.CloudflareConnections {
	return &cloudflareStore{db}
}

func (s *cloudflareStore) GetCloudflareConnection(ctx context.Context, target domain.TargetID) (domain.CloudflareConnection, error) {
	return builder.
		Query[domain.CloudflareConnection](`
		SELECT target_id, token, account_id, account_name, tunnel_allowed, mode, tunnel_id, tunnel_name, connected_at, connected_by
		FROM cloudflare_connections
		WHERE target_id = ?`, target).
		One(s.db, ctx, cloudflareConnectionMapper)
}

func (s *cloudflareStore) SaveCloudflareConnection(ctx context.Context, c domain.CloudflareConnection) error {
	return builder.
		Command(`
		INSERT INTO cloudflare_connections (target_id, token, account_id, account_name, tunnel_allowed, mode, tunnel_id, tunnel_name, connected_at, connected_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (target_id) DO UPDATE SET
			token = excluded.token,
			account_id = excluded.account_id,
			account_name = excluded.account_name,
			tunnel_allowed = excluded.tunnel_allowed,
			mode = excluded.mode,
			tunnel_id = excluded.tunnel_id,
			tunnel_name = excluded.tunnel_name,
			connected_at = excluded.connected_at,
			connected_by = excluded.connected_by`,
			c.Target, c.Token, c.AccountID, c.AccountName, c.TunnelAllowed, string(c.Mode), c.TunnelID, c.TunnelName, c.ConnectedAt, c.ConnectedBy).
		Exec(s.db, ctx)
}

func (s *cloudflareStore) DeleteCloudflareConnection(ctx context.Context, target domain.TargetID) error {
	return builder.
		Command("DELETE FROM cloudflare_connections WHERE target_id = ?", target).
		Exec(s.db, ctx)
}

func cloudflareConnectionMapper(s storage.Scanner) (c domain.CloudflareConnection, err error) {
	var mode string

	err = s.Scan(&c.Target, &c.Token, &c.AccountID, &c.AccountName, &c.TunnelAllowed, &mode, &c.TunnelID, &c.TunnelName, &c.ConnectedAt, &c.ConnectedBy)
	c.Mode = domain.CloudflareMode(mode)

	return c, err
}
