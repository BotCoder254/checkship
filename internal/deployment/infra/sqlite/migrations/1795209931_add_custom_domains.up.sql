-- Custom domains of application environments (seelf#5), per compose service, as JSON:
-- {"web": ["shop.example.com", "www.shop.example.com"]}. NULL when there are none.
ALTER TABLE apps ADD COLUMN production_domains TEXT NULL;
ALTER TABLE apps ADD COLUMN staging_domains TEXT NULL;

-- The domains a deployment was made with, frozen like its variables.
ALTER TABLE deployments ADD COLUMN config_domains TEXT NULL;

-- One row per custom domain: the primary key makes a domain serve a single application
-- environment, whatever happens in the code. Rows go with their application.
CREATE TABLE app_domains
(
    hostname TEXT NOT NULL,
    app_id TEXT NOT NULL,
    environment TEXT NOT NULL,
    service TEXT NOT NULL,

    CONSTRAINT pk_app_domains PRIMARY KEY (hostname),
    CONSTRAINT fk_app_domains_app_id FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_domains_app_id ON app_domains(app_id);

-- Revision of the proxy configuration each target was last configured with. When checkship
-- changes how it configures proxies, targets below the current revision are reconfigured once
-- at startup (the first change: certificates through the HTTP-01 challenge, which works
-- behind Cloudflare's proxy, instead of TLS-ALPN-01).
ALTER TABLE targets ADD COLUMN proxy_revision INTEGER NOT NULL DEFAULT 0;
