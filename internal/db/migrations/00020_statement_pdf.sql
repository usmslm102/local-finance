-- +goose Up
ALTER TABLE statement_imports ADD COLUMN source_pdf BLOB;

-- +goose Down
ALTER TABLE statement_imports DROP COLUMN source_pdf;
