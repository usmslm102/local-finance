-- +goose Up
CREATE TABLE splitwise_entries (
    id TEXT PRIMARY KEY,
    group_name TEXT NOT NULL,
    person TEXT NOT NULL,
    tx_date TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    cost_cents INTEGER NOT NULL CHECK(cost_cents > 0),
    net_cents INTEGER NOT NULL,
    share_cents INTEGER NOT NULL CHECK(share_cents >= 0 AND share_cents <= cost_cents),
    kind TEXT NOT NULL CHECK(kind IN ('EXPENSE','PAYMENT')),
    status TEXT NOT NULL CHECK(status IN ('REVIEW','CONFIRMED','IGNORED')),
    transaction_id TEXT UNIQUE REFERENCES transactions(id) ON DELETE SET NULL,
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL
);
CREATE INDEX idx_splitwise_date ON splitwise_entries(tx_date);

-- Read-only personal expense projection. Cash ledger, hashes and balances stay intact.
-- Columns are explicit so future statement schema changes cannot silently change the union.
CREATE VIEW personal_transactions AS
SELECT t.id,t.account_id,t.statement_import_id,t.tx_hash,t.tx_date,t.value_date,
       t.raw_narration,t.cleaned_payee,t.payment_mode,t.reference_number,t.tx_type,
       CASE WHEN s.kind='EXPENSE' THEN s.share_cents / 100.0 ELSE t.amount END AS amount,
       t.running_balance,t.category_id,t.is_recurring,t.notes,t.tags,t.created_at,
       t.upi_vpa,t.card_last4,t.is_transfer,
       CASE WHEN s.kind='PAYMENT' THEN 1 ELSE t.is_excluded END AS is_excluded,
       t.original_currency,t.original_amount,t.merchant_category,t.cashback_amount,
       t.reward_points_earned,t.transfer_peer_id,t.transfer_match_reason,t.net_amount,t.is_manual_category
FROM transactions t LEFT JOIN splitwise_entries s ON s.transaction_id=t.id AND s.status='CONFIRMED'
UNION ALL
SELECT s.id,'',NULL,s.id,s.tx_date,NULL,s.description,s.description,'OTHER','', 'DEBIT',
       s.share_cents / 100.0,NULL,
       COALESCE(s.category_id,(SELECT id FROM categories WHERE lower(name)=lower(s.category))),
       0,'Splitwise: ' || s.group_name,'',CURRENT_TIMESTAMP,NULL,NULL,0,0,NULL,NULL,NULL,0,0,NULL,NULL,NULL,0
FROM splitwise_entries s
WHERE s.status='CONFIRMED' AND s.kind='EXPENSE' AND s.transaction_id IS NULL AND s.share_cents>0;

-- +goose Down
DROP VIEW personal_transactions;
DROP TABLE splitwise_entries;
