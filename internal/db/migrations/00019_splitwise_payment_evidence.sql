-- +goose Up
ALTER TABLE splitwise_entries ADD COLUMN counterparty TEXT NOT NULL DEFAULT '';
-- Member patterns alone are no longer evidence of a Splitwise settlement.
DELETE FROM splitwise_entries WHERE id LIKE 'sw-settlement-%' AND status!='IGNORED';

-- +goose Down
ALTER TABLE splitwise_entries DROP COLUMN counterparty;
