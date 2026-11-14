-- Used by runners to find the jobs they can claim.
CREATE INDEX idx_scheduled_jobs_claim ON scheduled_jobs(retrieved, message_name, not_before);
