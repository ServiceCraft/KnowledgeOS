-- 013_pgvector.sql: RAG embeddings and indexing queue

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS kb_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('article', 'qa', 'pricing')),
    entity_id UUID NOT NULL,
    chunk_idx INT NOT NULL,
    theme_id UUID NULL,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    embedding vector(256) NOT NULL,
    model TEXT NOT NULL,
    dim INT NOT NULL DEFAULT 256 CHECK (dim > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(company_id, entity_type, entity_id, chunk_idx)
);

CREATE INDEX IF NOT EXISTS idx_kb_embeddings_vector_hnsw
    ON kb_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_kb_embeddings_company_entity
    ON kb_embeddings(company_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_kb_embeddings_company_theme
    ON kb_embeddings(company_id, theme_id);
CREATE INDEX IF NOT EXISTS idx_kb_embeddings_company_type
    ON kb_embeddings(company_id, entity_type);

CREATE TRIGGER update_kb_embeddings_updated_at
    BEFORE UPDATE ON kb_embeddings
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS kb_index_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('article', 'qa', 'pricing')),
    entity_id UUID NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('upsert', 'delete')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kb_index_jobs_claim
    ON kb_index_jobs(status, available_at, created_at);
CREATE INDEX IF NOT EXISTS idx_kb_index_jobs_company_status
    ON kb_index_jobs(company_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_index_jobs_pending_unique
    ON kb_index_jobs(company_id, entity_type, entity_id, operation)
    WHERE status IN ('pending', 'processing');

CREATE TRIGGER update_kb_index_jobs_updated_at
    BEFORE UPDATE ON kb_index_jobs
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
