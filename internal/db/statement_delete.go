package db

// StatementDeletion reports the exact rows removed by a committed deletion.
type StatementDeletion struct {
	StatementImportID   string `json:"statement_import_id"`
	AccountID           string `json:"account_id"`
	DeletedTransactions int64  `json:"deleted_transactions"`
	DeletedBills        int64  `json:"deleted_bills"`
}

// DeleteStatementImport removes only data currently owned by this upload.
// Duplicate upserts associate a transaction with its most recent import.
func (d *DB) DeleteStatementImport(id string) (*StatementDeletion, error) {
	result := &StatementDeletion{StatementImportID: id}
	err := d.WithStatementImport(func(w *StatementWriter) error {
		if err := w.conn.QueryRow(`SELECT account_id FROM statement_imports WHERE id = ?`, id).Scan(&result.AccountID); err != nil {
			return err
		}
		// Stop treating surviving counterparts as paired transfers before recategorizing them.
		if _, err := w.conn.Exec(`UPDATE transactions SET is_transfer = 0
   WHERE transfer_peer_id IN (SELECT id FROM transactions WHERE statement_import_id = ?)`, id); err != nil {
			return err
		}
		if _, err := w.reapplyRulesMatching(true, ` AND transfer_peer_id IN (SELECT id FROM transactions WHERE statement_import_id = ?) AND statement_import_id IS NOT ?`, id, id); err != nil {
			return err
		}
		if _, err := w.conn.Exec(`UPDATE transactions SET transfer_peer_id = NULL, transfer_match_reason = NULL WHERE transfer_peer_id IN (SELECT id FROM transactions WHERE statement_import_id = ?)`, id); err != nil {
			return err
		}
		deleted, err := w.conn.Exec(`DELETE FROM credit_card_bills WHERE statement_import_id = ?`, id)
		if err != nil {
			return err
		}
		if result.DeletedBills, err = deleted.RowsAffected(); err != nil {
			return err
		}
		deleted, err = w.conn.Exec(`DELETE FROM transactions WHERE statement_import_id = ?`, id)
		if err != nil {
			return err
		}
		if result.DeletedTransactions, err = deleted.RowsAffected(); err != nil {
			return err
		}
		if _, err := w.conn.Exec(`DELETE FROM statement_imports WHERE id = ?`, id); err != nil {
			return err
		}
		return w.RecalculateAccountBalance(result.AccountID)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
