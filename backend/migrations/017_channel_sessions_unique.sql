-- 017_channel_sessions_unique.sql: keep one active external chat session per channel.

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_sessions_company_channel_external_active
    ON chat_sessions(company_id, channel, external_chat_id)
    WHERE external_chat_id IS NOT NULL AND state <> 'closed';
