-- 016_chat_guardrails.sql: Б5 guardrails layer.
--
-- Adds machine-readable guardrail decisions to assistant messages (grounding,
-- confidence and refusal/escalation reasons), widens the session state machine
-- so a low-confidence answer can signal an operator handoff later, and stores
-- tunable per-tenant guardrail thresholds on bot_settings.

-- Assistant message guardrail outcome.
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS guardrail_action TEXT NOT NULL DEFAULT 'answer'
        CHECK (guardrail_action IN ('answer', 'refuse', 'escalate'));
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS confidence_score NUMERIC(4,3)
        CHECK (confidence_score IS NULL OR (confidence_score >= 0 AND confidence_score <= 1));
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS refusal_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS cited_source_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Widen the session state machine to leave room for the future handoff module.
-- Б5 only emits the escalation signal; the operator queue/UI lands later.
ALTER TABLE chat_sessions DROP CONSTRAINT IF EXISTS chat_sessions_state_check;
ALTER TABLE chat_sessions
    ADD CONSTRAINT chat_sessions_state_check
        CHECK (state IN ('bot', 'waiting_operator', 'operator', 'closed'));

-- Tunable, per-tenant guardrail thresholds. Defaults are conservative and
-- backward-compatible: thresholds of 0 disable the optional gates so existing
-- tenants keep current behavior until an admin opts in.
ALTER TABLE bot_settings
    ADD COLUMN IF NOT EXISTS min_retrieval_score NUMERIC(6,4) NOT NULL DEFAULT 0
        CHECK (min_retrieval_score >= 0);
ALTER TABLE bot_settings
    ADD COLUMN IF NOT EXISTS min_confidence NUMERIC(4,3) NOT NULL DEFAULT 0
        CHECK (min_confidence >= 0 AND min_confidence <= 1);
ALTER TABLE bot_settings
    ADD COLUMN IF NOT EXISTS allowed_theme_ids JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE bot_settings
    ADD COLUMN IF NOT EXISTS escalate_on_low_confidence BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE bot_settings
    ADD COLUMN IF NOT EXISTS require_citations BOOLEAN NOT NULL DEFAULT false;
