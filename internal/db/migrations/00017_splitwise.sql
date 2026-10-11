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
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    member_names TEXT NOT NULL DEFAULT '[]',
    notes TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    is_manual_category INTEGER NOT NULL DEFAULT 0,
    removed_transaction_id TEXT,
    counterparty TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_splitwise_date ON splitwise_entries(tx_date);

CREATE TABLE splitwise_groups (
 group_name TEXT PRIMARY KEY,
 person TEXT NOT NULL,
 member_names TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE splitwise_member_aliases (
 group_name TEXT NOT NULL,
 member_name TEXT NOT NULL,
 aliases TEXT NOT NULL DEFAULT '[]',
 pattern TEXT NOT NULL DEFAULT '',
 PRIMARY KEY (group_name, member_name)
);

-- Read-only projections preserve original bank cash movements and manual edits.
CREATE VIEW personal_transactions AS
SELECT t.id,t.account_id,t.statement_import_id,t.tx_hash,t.tx_date,t.value_date,
       t.raw_narration,t.cleaned_payee,t.payment_mode,t.reference_number,t.tx_type,
       CASE WHEN s.kind='EXPENSE' THEN s.share_cents / 100.0 ELSE t.amount END AS amount,
       t.running_balance,
       CASE WHEN s.kind='EXPENSE' AND COALESCE(t.is_manual_category,0)=0
            THEN COALESCE(s.category_id,(SELECT id FROM categories WHERE lower(name)=lower(s.category)),'cat_others') ELSE t.category_id END AS category_id,
       t.is_recurring,t.notes,t.tags,t.created_at,
       t.upi_vpa,t.card_last4,
       CASE WHEN s.kind='EXPENSE' THEN 0 ELSE t.is_transfer END AS is_transfer,
       CASE WHEN s.kind='PAYMENT' THEN 1 ELSE t.is_excluded END AS is_excluded,
       t.original_currency,t.original_amount,t.merchant_category,t.cashback_amount,
       t.reward_points_earned,t.transfer_peer_id,t.transfer_match_reason,t.net_amount,t.is_manual_category
FROM transactions t LEFT JOIN splitwise_entries s ON s.transaction_id=t.id AND s.status='CONFIRMED'
UNION ALL
SELECT s.id,'',NULL,s.id,s.tx_date,NULL,s.description,s.description,'OTHER','', 'DEBIT',
       s.share_cents / 100.0,NULL,
       CASE WHEN s.is_manual_category=1 THEN s.category_id ELSE COALESCE(s.category_id,(SELECT id FROM categories WHERE lower(name)=lower(s.category))) END,
       0,COALESCE(NULLIF(s.notes,''),'Splitwise: ' || s.group_name),s.tags,CURRENT_TIMESTAMP,NULL,NULL,0,0,NULL,NULL,NULL,0,0,NULL,NULL,NULL,s.is_manual_category
FROM splitwise_entries s
WHERE s.status='CONFIRMED' AND s.kind='EXPENSE' AND s.transaction_id IS NULL AND s.share_cents>0;
CREATE VIEW ledger_transactions AS
SELECT t.id,t.account_id,t.statement_import_id,t.tx_hash,t.tx_date,t.value_date,t.raw_narration,t.cleaned_payee,t.payment_mode,t.reference_number,t.tx_type,t.amount,t.running_balance,CASE WHEN s.kind='PAYMENT' THEN 'cat_transfers' ELSE p.category_id END AS category_id,t.is_recurring,t.notes,t.tags,t.created_at,t.upi_vpa,t.card_last4,CASE WHEN s.kind='PAYMENT' THEN 1 ELSE t.is_transfer END AS is_transfer,t.is_excluded,t.original_currency,t.original_amount,t.merchant_category,t.cashback_amount,t.reward_points_earned,t.transfer_peer_id,t.transfer_match_reason,t.net_amount,t.is_manual_category FROM transactions t JOIN personal_transactions p ON p.id=t.id AND p.account_id!='' LEFT JOIN splitwise_entries s ON s.transaction_id=t.id AND s.status='CONFIRMED'
UNION ALL
SELECT id,account_id,statement_import_id,tx_hash,tx_date,value_date,raw_narration,cleaned_payee,payment_mode,reference_number,tx_type,amount,running_balance,category_id,is_recurring,notes,tags,created_at,upi_vpa,card_last4,is_transfer,is_excluded,original_currency,original_amount,merchant_category,cashback_amount,reward_points_earned,transfer_peer_id,transfer_match_reason,net_amount,is_manual_category FROM personal_transactions WHERE account_id='';

-- +goose Down
DROP VIEW ledger_transactions;
DROP VIEW personal_transactions;
DROP TABLE splitwise_member_aliases;
DROP TABLE splitwise_groups;
DROP TABLE splitwise_entries;
