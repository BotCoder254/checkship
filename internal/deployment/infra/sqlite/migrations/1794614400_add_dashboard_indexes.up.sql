-- Indexes for the Overview dashboard: ranges and the activity feed read deployments by request
-- time across every application (keyset order: requested_at DESC, app_id DESC, deployment_number DESC).
CREATE INDEX IF NOT EXISTS idx_deployments_requested_at ON deployments(requested_at, app_id, deployment_number);
CREATE INDEX IF NOT EXISTS idx_deployments_environment_requested_at ON deployments(config_environment, requested_at);
