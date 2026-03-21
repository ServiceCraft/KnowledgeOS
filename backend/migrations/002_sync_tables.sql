-- 002_sync_tables.sql: Sync infrastructure

-- Monotonic sequence per company for sync versioning
CREATE TABLE sync_sequence (
    company_id  UUID PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    current_seq BIGINT NOT NULL DEFAULT 0
);

-- Watermarks track last synced positions
CREATE TABLE sync_watermarks (
    company_id     UUID PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_local_seq BIGINT NOT NULL DEFAULT 0,
    last_cloud_seq BIGINT NOT NULL DEFAULT 0,
    last_sync_at   TIMESTAMPTZ
);

-- Current sync status per company
CREATE TABLE sync_status (
    company_id          UUID PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    last_sync_at        TIMESTAMPTZ,
    last_sync_result    TEXT,
    last_error          TEXT,
    subscription_active BOOLEAN NOT NULL DEFAULT false
);

-- Sync audit log
CREATE TABLE sync_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    direction     TEXT NOT NULL CHECK (direction IN ('push', 'pull')),
    entity_type   TEXT NOT NULL,
    entity_id     UUID NOT NULL,
    seq           BIGINT NOT NULL,
    status        TEXT NOT NULL CHECK (status IN ('ok', 'conflict', 'error')),
    conflict_note TEXT,
    synced_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sync_log_company ON sync_log(company_id, synced_at DESC);
