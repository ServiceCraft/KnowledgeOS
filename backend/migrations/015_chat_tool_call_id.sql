-- 015_chat_tool_call_id.sql: track tool_call_id on chat messages for the Б4
-- tool-calling loop, so role=tool messages can be linked back to the assistant
-- tool call that produced them when reconstructing LLM history.

ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS tool_call_id TEXT NOT NULL DEFAULT '';
