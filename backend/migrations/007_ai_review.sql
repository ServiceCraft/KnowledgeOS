-- AI answer review workflow
ALTER TABLE qa_pairs ADD COLUMN IF NOT EXISTS ai_answer TEXT;
ALTER TABLE qa_pairs ADD COLUMN IF NOT EXISTS ai_status TEXT
    CHECK (ai_status IN ('pending', 'accepted', 'rejected', 'edited'));
ALTER TABLE qa_pairs ADD COLUMN IF NOT EXISTS ai_reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE qa_pairs ADD COLUMN IF NOT EXISTS ai_reviewed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_qa_ai_status ON qa_pairs(company_id, ai_status)
    WHERE ai_status IS NOT NULL AND deleted_at IS NULL;
