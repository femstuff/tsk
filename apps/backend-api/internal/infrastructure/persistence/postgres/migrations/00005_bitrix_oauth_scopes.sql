-- +goose Up
ALTER TABLE bitrix_oauth_sessions
    ADD COLUMN IF NOT EXISTS oauth_scopes TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bitrix_oauth_sessions DROP COLUMN IF EXISTS oauth_scopes;
