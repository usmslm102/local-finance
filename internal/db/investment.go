package db

import (
	"encoding/json"
	"local-finance/internal/models"
)

// SaveInvestmentSnapshot is idempotent for identical files. Each statement is
// a full snapshot, so callers must never add holdings from different dates.
func (d *DB) SaveInvestmentSnapshot(snapshot *models.InvestmentSnapshot, hash string) (*models.InvestmentSnapshot, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, false, err
	}
	res, err := d.conn.Exec(`INSERT INTO investment_snapshots(id,provider,account_ref,as_of,file_hash,imported_at,data_json) VALUES(?,?,?,?,?,?,?) ON CONFLICT(file_hash) DO NOTHING`, snapshot.ID, snapshot.Provider, snapshot.AccountRef, snapshot.AsOf, hash, snapshot.ImportedAt, string(data))
	if err != nil {
		return nil, false, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	if count == 0 {
		var existing string
		if err := d.conn.QueryRow(`SELECT data_json FROM investment_snapshots WHERE file_hash=?`, hash).Scan(&existing); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal([]byte(existing), snapshot); err != nil {
			return nil, false, err
		}
	}
	return snapshot, count == 0, nil
}

func (d *DB) ListInvestmentSnapshots() ([]models.InvestmentSnapshot, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listInvestmentSnapshots()
}

// GetInvestmentSnapshot loads one portfolio without decoding unrelated history.
func (d *DB) GetInvestmentSnapshot(id string) (*models.InvestmentSnapshot, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var data string
	if err := d.conn.QueryRow(`SELECT data_json FROM investment_snapshots WHERE id = ?`, id).Scan(&data); err != nil {
		return nil, err
	}
	var snapshot models.InvestmentSnapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// InvestmentPortfolioKey uses the earliest stored snapshot's random ID instead
// of a reversible digest of a short broker reference. It remains stable while
// that snapshot exists, and requires no additional portfolio storage.
func (d *DB) InvestmentPortfolioKey(snapshot *models.InvestmentSnapshot) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var key string
	err := d.conn.QueryRow(`SELECT id FROM investment_snapshots
		WHERE provider = ? AND account_ref = ? AND json_extract(data_json, '$.currency') = ?
		ORDER BY rowid ASC LIMIT 1`, snapshot.Provider, snapshot.AccountRef, snapshot.Currency).Scan(&key)
	return key, err
}

// listInvestmentSnapshots requires the caller to hold d.mu. Export already
// holds a read lock, so it must not recursively acquire it with a writer waiting.
func (d *DB) listInvestmentSnapshots() ([]models.InvestmentSnapshot, error) {
	rows, err := d.conn.Query(`SELECT data_json FROM investment_snapshots ORDER BY as_of DESC, imported_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.InvestmentSnapshot{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var snapshot models.InvestmentSnapshot
		if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	return result, rows.Err()
}

func (d *DB) DeleteInvestmentSnapshot(id string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.conn.Exec(`DELETE FROM investment_snapshots WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	return count > 0, err
}
