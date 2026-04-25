-- 008_calls.sql: Calls (with full transcript) and per-QA mentions

CREATE TABLE calls (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    external_id   TEXT,
    title         TEXT NOT NULL DEFAULT '',
    occurred_at   TIMESTAMPTZ,
    duration_sec  INTEGER,
    audio_url     TEXT,
    transcript    TEXT NOT NULL DEFAULT '',
    sync_version  BIGINT NOT NULL DEFAULT 0,
    sync_origin   TEXT NOT NULL DEFAULT '',
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_calls_company  ON calls(company_id);
CREATE INDEX idx_calls_occurred ON calls(company_id, occurred_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_calls_sync     ON calls(company_id, sync_version)     WHERE deleted_at IS NULL;
CREATE INDEX idx_calls_deleted  ON calls(deleted_at)                   WHERE deleted_at IS NOT NULL;
CREATE UNIQUE INDEX idx_calls_external
    ON calls(company_id, external_id)
    WHERE external_id IS NOT NULL AND deleted_at IS NULL;

-- Mentions: a span of a call's transcript that corresponds to a QA pair.
-- start_offset/end_offset are JS-style (UTF-16 code unit) indices into calls.transcript;
-- snippet is the denormalized excerpt so the list endpoint avoids JOIN ... LATERAL substr().
CREATE TABLE qa_pair_call_mentions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    qa_pair_id    UUID NOT NULL REFERENCES qa_pairs(id) ON DELETE CASCADE,
    call_id       UUID NOT NULL REFERENCES calls(id)    ON DELETE CASCADE,
    snippet       TEXT NOT NULL,
    start_offset  INTEGER NOT NULL,
    end_offset    INTEGER NOT NULL,
    start_sec     NUMERIC(10,3),
    end_sec       NUMERIC(10,3),
    confidence    NUMERIC(4,3),
    sync_version  BIGINT NOT NULL DEFAULT 0,
    sync_origin   TEXT NOT NULL DEFAULT '',
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT mention_offsets_valid CHECK (start_offset >= 0 AND end_offset > start_offset)
);

CREATE INDEX idx_mentions_company ON qa_pair_call_mentions(company_id);
CREATE INDEX idx_mentions_qa      ON qa_pair_call_mentions(company_id, qa_pair_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_mentions_call    ON qa_pair_call_mentions(company_id, call_id)    WHERE deleted_at IS NULL;
CREATE INDEX idx_mentions_sync    ON qa_pair_call_mentions(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_mentions_deleted ON qa_pair_call_mentions(deleted_at) WHERE deleted_at IS NOT NULL;
