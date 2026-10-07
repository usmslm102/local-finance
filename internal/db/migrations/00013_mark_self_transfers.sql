-- +goose Up
-- Older payment-specific narration parsing could miss this explicit marker.
UPDATE transactions
SET is_transfer = 1
WHERE is_transfer = 0 AND INSTR(UPPER(raw_narration), 'SELF TRANSFER') > 0;

-- +goose Down
-- Keep corrected flags: clearing them could undo user-confirmed transfers.
SELECT 1;
