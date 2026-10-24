-- Allow AGY (Antigravity) as an agent account provider.
-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
PRAGMA writable_schema = ON;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE sqlite_master
SET sql = replace(sql, 'CHECK (provider IN (''claude'', ''codex''))', 'CHECK (provider IN (''claude'', ''codex'', ''agy''))')
WHERE type = 'table' AND name IN ('agent_accounts', 'agent_account_defaults', 'session_agent_accounts')
  AND instr(sql, 'CHECK (provider IN (''claude'', ''codex''))') > 0;
-- +goose StatementEnd
-- +goose StatementBegin
PRAGMA writable_schema = RESET;
-- +goose StatementEnd
-- +goose StatementBegin
INSERT OR IGNORE INTO session_agent_accounts (session_id, provider, account_id, bound_at)
SELECT sessions.id, 'agy', 'device-agy', sessions.created_at
FROM sessions;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM session_agent_accounts WHERE provider = 'agy';
-- +goose StatementEnd
-- +goose StatementBegin
DELETE FROM agent_account_defaults WHERE provider = 'agy';
-- +goose StatementEnd
-- +goose StatementBegin
DELETE FROM agent_accounts WHERE provider = 'agy';
-- +goose StatementEnd
-- +goose StatementBegin
PRAGMA writable_schema = ON;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE sqlite_master
SET sql = replace(sql, 'CHECK (provider IN (''claude'', ''codex'', ''agy''))', 'CHECK (provider IN (''claude'', ''codex''))')
WHERE type = 'table' AND name IN ('agent_accounts', 'agent_account_defaults', 'session_agent_accounts')
  AND instr(sql, 'CHECK (provider IN (''claude'', ''codex'', ''agy''))') > 0;
-- +goose StatementEnd
-- +goose StatementBegin
PRAGMA writable_schema = RESET;
-- +goose StatementEnd
