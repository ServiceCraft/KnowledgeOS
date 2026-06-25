-- 011_bot_settings.sql: per-tenant bot configuration

CREATE TABLE bot_settings (
    company_id      UUID PRIMARY KEY REFERENCES companies(id) ON DELETE CASCADE,
    enabled         BOOLEAN NOT NULL DEFAULT false,
    provider        TEXT NOT NULL DEFAULT 'yandex' CHECK (provider IN ('yandex')),
    model_tier      TEXT NOT NULL DEFAULT 'lite' CHECK (model_tier IN ('lite', 'pro')),
    model           TEXT,
    temperature     NUMERIC(3,2) NOT NULL DEFAULT 0.20 CHECK (temperature >= 0 AND temperature <= 2),
    max_tokens      INT NOT NULL DEFAULT 1024 CHECK (max_tokens > 0 AND max_tokens <= 8192),
    persona_name    TEXT NOT NULL DEFAULT 'Администратор',
    persona_tone    TEXT NOT NULL DEFAULT 'friendly_professional',
    persona_rules   TEXT NOT NULL DEFAULT '',
    enabled_modules JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_bot_settings_updated_at
BEFORE UPDATE ON bot_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
