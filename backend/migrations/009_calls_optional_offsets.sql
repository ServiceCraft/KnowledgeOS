-- Allow mentions without precise transcript offsets (e.g. paraphrased AI snippets).
-- The pair must be either both NULL or both valid; partial offsets are still rejected.

ALTER TABLE qa_pair_call_mentions ALTER COLUMN start_offset DROP NOT NULL;
ALTER TABLE qa_pair_call_mentions ALTER COLUMN end_offset   DROP NOT NULL;

ALTER TABLE qa_pair_call_mentions DROP CONSTRAINT IF EXISTS mention_offsets_valid;
ALTER TABLE qa_pair_call_mentions ADD CONSTRAINT mention_offsets_valid CHECK (
    (start_offset IS NULL AND end_offset IS NULL)
    OR (start_offset >= 0 AND end_offset > start_offset)
);
