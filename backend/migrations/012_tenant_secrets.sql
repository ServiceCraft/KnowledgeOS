-- 012_tenant_secrets.sql: encrypted per-tenant secrets

CREATE TABLE tenant_secrets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('llm', 'telegram', 'max', 'vk', 'bitrix24')),
    ciphertext BYTEA NOT NULL,
    nonce      BYTEA NOT NULL,
    metadata   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (company_id, kind)
);

CREATE INDEX idx_tenant_secrets_company ON tenant_secrets(company_id);

CREATE TRIGGER trg_tenant_secrets_updated_at
BEFORE UPDATE ON tenant_secrets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
