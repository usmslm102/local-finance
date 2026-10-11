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

// InvestmentSnapshotWithPortfolio couples a snapshot with its grouping identity
// from the same read, so deletion cannot change keys midway through a response.
type InvestmentSnapshotWithPortfolio struct {
	Snapshot     models.InvestmentSnapshot
	PortfolioKey string
}

const investmentSnapshotsWithPortfolio = `SELECT id, data_json,
 FIRST_VALUE(id) OVER (PARTITION BY provider, account_ref, json_extract(data_json, '$.currency') ORDER BY rowid) AS portfolio_key,
 as_of, imported_at FROM investment_snapshots`

func (d *DB) ListInvestmentSnapshotsWithPortfolio() ([]InvestmentSnapshotWithPortfolio, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.conn.Query("SELECT data_json, portfolio_key FROM (" + investmentSnapshotsWithPortfolio + ") ORDER BY as_of DESC, imported_at DESC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []InvestmentSnapshotWithPortfolio{}
	for rows.Next() {
		var item InvestmentSnapshotWithPortfolio
		var data string
		if err := rows.Scan(&data, &item.PortfolioKey); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(data), &item.Snapshot); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *DB) GetInvestmentSnapshotWithPortfolio(id string) (*InvestmentSnapshotWithPortfolio, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var item InvestmentSnapshotWithPortfolio
	var data string
	if err := d.conn.QueryRow("SELECT data_json, portfolio_key FROM ("+investmentSnapshotsWithPortfolio+") WHERE id = ?", id).Scan(&data, &item.PortfolioKey); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(data), &item.Snapshot); err != nil {
		return nil, err
	}
	return &item, nil
}
