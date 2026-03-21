-- 001_init.sql: Core tables for KnowledgeOS

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Companies
CREATE TABLE companies (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    tier       TEXT NOT NULL DEFAULT 'local' CHECK (tier IN ('local', 'cloud')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Users
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID REFERENCES companies(id) ON DELETE CASCADE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('superadmin', 'admin', 'editor', 'viewer')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_company ON users(company_id);

-- Themes
CREATE TABLE themes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT,
    sync_version BIGINT NOT NULL DEFAULT 0,
    sync_origin  TEXT NOT NULL DEFAULT '',
    created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_themes_company ON themes(company_id);
CREATE INDEX idx_themes_sync ON themes(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_themes_deleted ON themes(deleted_at) WHERE deleted_at IS NOT NULL;

-- QA Pairs
CREATE TABLE qa_pairs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    theme_id      UUID REFERENCES themes(id) ON DELETE SET NULL,
    question      TEXT NOT NULL,
    answer        TEXT NOT NULL,
    is_faq        BOOLEAN NOT NULL DEFAULT false,
    is_locked     BOOLEAN NOT NULL DEFAULT false,
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(question, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(answer, '')), 'B')
    ) STORED,
    sync_version  BIGINT NOT NULL DEFAULT 0,
    sync_origin   TEXT NOT NULL DEFAULT '',
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_qa_company ON qa_pairs(company_id);
CREATE INDEX idx_qa_theme ON qa_pairs(company_id, theme_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_qa_faq ON qa_pairs(company_id) WHERE is_faq = true AND deleted_at IS NULL;
CREATE INDEX idx_qa_sync ON qa_pairs(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_qa_deleted ON qa_pairs(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_qa_search ON qa_pairs USING GIN(search_vector);

-- Pricing Nodes
CREATE TABLE pricing_nodes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    parent_id    UUID REFERENCES pricing_nodes(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    node_type    TEXT NOT NULL CHECK (node_type IN ('category', 'service', 'option')),
    name         TEXT NOT NULL,
    price        NUMERIC(12,2),
    sync_version BIGINT NOT NULL DEFAULT 0,
    sync_origin  TEXT NOT NULL DEFAULT '',
    created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_pricing_company ON pricing_nodes(company_id);
CREATE INDEX idx_pricing_parent ON pricing_nodes(company_id, parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_sync ON pricing_nodes(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_deleted ON pricing_nodes(deleted_at) WHERE deleted_at IS NOT NULL;

-- Articles
CREATE TABLE articles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    body          TEXT NOT NULL DEFAULT '',
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body, '')), 'B')
    ) STORED,
    sync_version  BIGINT NOT NULL DEFAULT 0,
    sync_origin   TEXT NOT NULL DEFAULT '',
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_articles_company ON articles(company_id);
CREATE INDEX idx_articles_sync ON articles(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_articles_deleted ON articles(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_articles_search ON articles USING GIN(search_vector);

-- Comments (polymorphic)
CREATE TABLE comments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type  TEXT NOT NULL CHECK (entity_type IN ('qa', 'article', 'pricing')),
    entity_id    UUID NOT NULL,
    body         TEXT NOT NULL,
    author_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    sync_version BIGINT NOT NULL DEFAULT 0,
    sync_origin  TEXT NOT NULL DEFAULT '',
    created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_comments_company ON comments(company_id);
CREATE INDEX idx_comments_entity ON comments(company_id, entity_type, entity_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_sync ON comments(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_deleted ON comments(deleted_at) WHERE deleted_at IS NOT NULL;

-- Entity Links (polymorphic, internal or external)
CREATE TABLE entity_links (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    source_type  TEXT NOT NULL CHECK (source_type IN ('qa', 'article', 'pricing')),
    source_id    UUID NOT NULL,
    target_type  TEXT CHECK (target_type IN ('qa', 'article', 'pricing')),
    target_id    UUID,
    url          TEXT,
    label        TEXT,
    sync_version BIGINT NOT NULL DEFAULT 0,
    sync_origin  TEXT NOT NULL DEFAULT '',
    created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    CONSTRAINT internal_or_external CHECK (
        (target_type IS NOT NULL AND target_id IS NOT NULL AND url IS NULL)
        OR
        (target_type IS NULL AND target_id IS NULL AND url IS NOT NULL)
    )
);

CREATE INDEX idx_links_company ON entity_links(company_id);
CREATE INDEX idx_links_source ON entity_links(company_id, source_type, source_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_links_sync ON entity_links(company_id, sync_version) WHERE deleted_at IS NULL;
CREATE INDEX idx_links_deleted ON entity_links(deleted_at) WHERE deleted_at IS NOT NULL;
