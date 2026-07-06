-- 017_channel_webhook_events.sql: idempotency keys for channel webhooks.

CREATE TABLE IF NOT EXISTS channel_webhook_events (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    channel TEXT NOT NULL CHECK (channel IN ('telegram', 'max', 'vk')),
    update_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (company_id, channel, update_id)
);

CREATE INDEX IF NOT EXISTS idx_channel_webhook_events_created
    ON channel_webhook_events(created_at);
