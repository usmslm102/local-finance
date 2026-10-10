package db

import "local-finance/internal/models"

func (d *DB) ListSplitwiseMappings() ([]models.SplitwiseMapping, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entries, err := d.listSplitwise()
	if err != nil {
		return nil, err
	}
	rows, err := d.conn.Query(`SELECT s.id,t.id,t.tx_date,t.raw_narration,COALESCE(t.cleaned_payee,''),t.amount,t.tx_type,COALESCE(NULLIF(a.nickname,''),a.bank_name) FROM splitwise_entries s JOIN transactions t ON t.id=s.transaction_id JOIN accounts a ON a.id=t.account_id WHERE s.status='CONFIRMED'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	bank := map[string]*models.Transaction{}
	for rows.Next() {
		var id string
		var t models.Transaction
		if err := rows.Scan(&id, &t.ID, &t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.Amount, &t.TxType, &t.AccountName); err != nil {
			return nil, err
		}
		bank[id] = &t
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := []models.SplitwiseMapping{}
	for _, e := range entries {
		if e.Status == "CONFIRMED" {
			result = append(result, models.SplitwiseMapping{Entry: e, Bank: bank[e.ID]})
		}
	}
	return result, nil
}
