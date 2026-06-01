-- +goose Up
ALTER TABLE document_jobs
    ADD COLUMN IF NOT EXISTS bitrix_deal_id INT,
    ADD COLUMN IF NOT EXISTS bitrix_deal_title TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_document_jobs_bitrix_deal_id
    ON document_jobs (bitrix_deal_id)
    WHERE bitrix_deal_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_document_jobs_mobile_requested
    ON document_jobs (requested_by, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_document_jobs_mobile_requested;
DROP INDEX IF EXISTS idx_document_jobs_bitrix_deal_id;
ALTER TABLE document_jobs
    DROP COLUMN IF EXISTS bitrix_deal_title,
    DROP COLUMN IF EXISTS bitrix_deal_id;
