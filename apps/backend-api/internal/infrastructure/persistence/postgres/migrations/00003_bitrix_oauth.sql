-- +goose Up
CREATE TABLE IF NOT EXISTS bitrix_oauth_sessions (
    id TEXT PRIMARY KEY,
    state TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    portal_domain TEXT NOT NULL,
    bitrix_user_id INT NOT NULL DEFAULT 0,
    user_name TEXT NOT NULL DEFAULT '',
    access_token TEXT NOT NULL DEFAULT '',
    refresh_token TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bitrix_oauth_sessions_status
    ON bitrix_oauth_sessions (status, updated_at DESC);

-- +goose Down
DROP TABLE IF EXISTS bitrix_oauth_sessions;
