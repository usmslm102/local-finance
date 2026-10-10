-- +goose Up
ALTER TABLE mcp_settings ADD COLUMN allow_categorization_writes BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE mcp_settings DROP COLUMN allow_categorization_writes;
