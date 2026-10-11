-- +goose Up
CREATE TABLE statement_import_transactions (
    statement_import_id TEXT NOT NULL REFERENCES statement_imports(id) ON DELETE CASCADE,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    PRIMARY KEY (statement_import_id, transaction_id)
);
CREATE INDEX idx_statement_records_transaction ON statement_import_transactions(transaction_id);

-- Only the latest association is available for historical imports.
INSERT INTO statement_import_transactions (statement_import_id, transaction_id)
SELECT t.statement_import_id, t.id FROM transactions t
JOIN statement_imports s ON s.id = t.statement_import_id;

-- +goose Down
DROP TABLE statement_import_transactions;
