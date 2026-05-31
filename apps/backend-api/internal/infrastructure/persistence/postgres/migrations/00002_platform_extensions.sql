-- +goose Up
ALTER TABLE source_documents ALTER COLUMN template_id DROP NOT NULL;

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_secret BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS stored_files (
    id TEXT PRIMARY KEY,
    namespace TEXT NOT NULL,
    storage_key TEXT NOT NULL UNIQUE,
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256_hex TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stored_files_entity
    ON stored_files (entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_stored_files_namespace_created
    ON stored_files (namespace, created_at DESC);

CREATE TABLE IF NOT EXISTS transcriptions (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    source_document_id TEXT REFERENCES source_documents(id),
    job_id TEXT REFERENCES document_jobs(id),
    transcript TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT '',
    whisper_model TEXT NOT NULL DEFAULT '',
    duration_ms INT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_transcriptions_job_id
    ON transcriptions (job_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transcriptions_source_document_id
    ON transcriptions (source_document_id);

CREATE TABLE IF NOT EXISTS job_processing_attempts (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES document_jobs(id),
    attempt_no INT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_job_processing_attempts_job_attempt
    ON job_processing_attempts (job_id, attempt_no);

CREATE INDEX IF NOT EXISTS idx_job_processing_attempts_job_started
    ON job_processing_attempts (job_id, started_at DESC);

-- +goose Down
DROP TABLE IF EXISTS job_processing_attempts;
DROP TABLE IF EXISTS transcriptions;
DROP TABLE IF EXISTS stored_files;
DROP TABLE IF EXISTS app_settings;
