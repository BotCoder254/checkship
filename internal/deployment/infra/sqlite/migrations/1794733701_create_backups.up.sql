-- Backups outlive their application and target, so app_id and target_id have no
-- foreign key: a backup of a deleted app can still be downloaded or restored.
CREATE TABLE backups (
    id TEXT NOT NULL
    ,app_id TEXT NOT NULL
    ,app_name TEXT NOT NULL
    ,environment TEXT NOT NULL
    ,target_id TEXT NOT NULL
    ,reason TEXT NOT NULL
    ,status TEXT NOT NULL
    ,stop_containers INTEGER NOT NULL
    ,volumes TEXT NOT NULL
    ,result TEXT NULL
    ,local_copy INTEGER NOT NULL DEFAULT 0
    ,stored_copy INTEGER NOT NULL DEFAULT 0
    ,store_error TEXT NULL
    ,error_code TEXT NULL
    ,requested_at DATETIME NOT NULL
    ,requested_by TEXT NOT NULL
    ,finished_at DATETIME NULL
    ,CONSTRAINT pk_backups PRIMARY KEY(id)
    ,CONSTRAINT fk_backups_requested_by FOREIGN KEY(requested_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_backups_app_requested_at ON backups(app_id, requested_at DESC);
CREATE INDEX idx_backups_requested_at ON backups(requested_at DESC);

-- One row per restore, so its outcome stays visible once the job is gone.
CREATE TABLE backup_restores (
    id TEXT NOT NULL
    ,backup_id TEXT NOT NULL
    ,app_id TEXT NOT NULL
    ,environment TEXT NOT NULL
    ,target_id TEXT NOT NULL
    ,volumes TEXT NOT NULL
    ,snapshot_first INTEGER NOT NULL
    ,status TEXT NOT NULL
    ,restored TEXT NOT NULL DEFAULT '[]'
    ,snapshot_id TEXT NULL
    ,error_code TEXT NULL
    ,requested_at DATETIME NOT NULL
    ,requested_by TEXT NOT NULL
    ,finished_at DATETIME NULL
    ,CONSTRAINT pk_backup_restores PRIMARY KEY(id)
    ,CONSTRAINT fk_backup_restores_requested_by FOREIGN KEY(requested_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_backup_restores_backup ON backup_restores(backup_id, requested_at DESC);
CREATE INDEX idx_backup_restores_app ON backup_restores(app_id, requested_at DESC);

-- Where backups are copied besides the data folder: one row for the whole instance.
CREATE TABLE backup_storage (
    id INTEGER NOT NULL
    ,mode TEXT NOT NULL
    ,target_id TEXT NULL
    ,endpoint TEXT NOT NULL DEFAULT ''
    ,use_ssl INTEGER NOT NULL DEFAULT 0
    ,path_style INTEGER NOT NULL DEFAULT 1
    ,region TEXT NOT NULL DEFAULT ''
    ,bucket TEXT NOT NULL DEFAULT ''
    ,prefix TEXT NOT NULL DEFAULT ''
    ,access_key TEXT NOT NULL DEFAULT ''
    ,secret_key TEXT NOT NULL DEFAULT ''
    ,keep_local INTEGER NOT NULL DEFAULT 1
    ,status TEXT NOT NULL
    ,error_code TEXT NULL
    ,version DATETIME NOT NULL
    ,CONSTRAINT pk_backup_storage PRIMARY KEY(id)
    ,CONSTRAINT single_backup_storage CHECK(id = 1)
);
