-- Connect Cloudflare: the token a target uses to set up DNS records and a tunnel in the
-- person's Cloudflare account. The token is a secret, never returned by the API. Rows go with
-- their target.
CREATE TABLE cloudflare_connections
(
    target_id TEXT NOT NULL,
    token TEXT NOT NULL,
    account_id TEXT NOT NULL,
    account_name TEXT NOT NULL,
    tunnel_allowed INTEGER NOT NULL DEFAULT 0,
    mode TEXT NOT NULL DEFAULT 'tunnel',
    tunnel_id TEXT NULL,
    tunnel_name TEXT NULL,
    connected_at DATETIME NOT NULL,
    connected_by TEXT NOT NULL,

    CONSTRAINT pk_cloudflare_connections PRIMARY KEY (target_id),
    CONSTRAINT fk_cloudflare_connections_target_id FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE,
    CONSTRAINT fk_cloudflare_connections_connected_by FOREIGN KEY (connected_by) REFERENCES users(id) ON DELETE CASCADE
);
