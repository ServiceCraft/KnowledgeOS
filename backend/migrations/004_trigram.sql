-- 004_trigram.sql: Enable pg_trgm for trigram search

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram indexes for fuzzy search
CREATE INDEX idx_qa_question_trgm ON qa_pairs USING GIN(question gin_trgm_ops);
CREATE INDEX idx_qa_answer_trgm ON qa_pairs USING GIN(answer gin_trgm_ops);
CREATE INDEX idx_articles_title_trgm ON articles USING GIN(title gin_trgm_ops);
CREATE INDEX idx_articles_body_trgm ON articles USING GIN(body gin_trgm_ops);
