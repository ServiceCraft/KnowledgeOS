-- 021_channel_webhook_events_status.sql: track processing lifecycle for channel
-- webhook idempotency so a transient failure does not permanently drop a message.
-- Existing rows were recorded only after successful processing, so they default to
-- 'done'; new rows are inserted as 'processing' and promoted to 'done' only after
-- the reply is delivered.

ALTER TABLE channel_webhook_events
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'done'
        CHECK (status IN ('processing', 'done', 'failed'));
ALTER TABLE channel_webhook_events
    ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0);
ALTER TABLE channel_webhook_events
    ADD COLUMN IF NOT EXISTS last_error TEXT;
ALTER TABLE channel_webhook_events
    ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;
ALTER TABLE channel_webhook_events
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_channel_webhook_events_status
    ON channel_webhook_events(status, updated_at);
