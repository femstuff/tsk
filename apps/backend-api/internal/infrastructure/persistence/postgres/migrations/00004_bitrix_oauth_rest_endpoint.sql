-- +goose Up
ALTER TABLE bitrix_oauth_sessions
    ADD COLUMN IF NOT EXISTS rest_endpoint TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE bitrix_oauth_sessions DROP COLUMN IF EXISTS rest_endpoint;
