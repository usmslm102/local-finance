package db

import (
	"database/sql"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"local-finance/internal/models"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type DB struct {
	mu   sync.RWMutex
	conn *sql.DB
	path string
}

func NewDB(dbPath string) (*DB, error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		absPath = dbPath
	}

	conn, err := sql.Open("sqlite", absPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite works best with 1 writer connection

	d := &DB{conn: conn, path: absPath}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Restrict database file permissions to 0600 (owner read-write only) for local SQLite protection
	_ = os.Chmod(absPath, 0600)
	_ = os.Chmod(absPath+"-wal", 0600)
	_ = os.Chmod(absPath+"-shm", 0600)

	if err := d.seedDefaultCategories(); err != nil {
		return nil, fmt.Errorf("failed to seed default categories: %w", err)
	}

	if err := d.seedDefaultRules(); err != nil {
		return nil, fmt.Errorf("failed to seed default rules: %w", err)
	}

	return d, nil
}

func (d *DB) ResetDatabase() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.clearMCPAccess(); err != nil {
		return err
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM subscriptions`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM credit_card_bills`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM transactions`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM statement_imports`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM accounts`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_ = d.seedDefaultCategories()
	_ = d.seedDefaultRules()

	return nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(d.conn, "migrations"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func (d *DB) seedDefaultCategories() error {
	defaultCategories := []struct {
		ID       string
		Name     string
		ColorHex string
		Icon     string
	}{
		{"cat_food", "Food & Dining", "#EF4444", "Utensils"},
		{"cat_groceries", "Groceries & Essentials", "#10B981", "ShoppingBag"},
		{"cat_shopping", "Shopping & E-Commerce", "#F59E0B", "ShoppingCart"},
		{"cat_bills", "Bills & Utilities", "#3B82F6", "Receipt"},
		{"cat_travel", "Fuel & Transport", "#6366F1", "Car"},
		{"cat_invest", "Investments & Savings", "#059669", "TrendingUp"},
		{"cat_entertainment", "Entertainment & OTT", "#EC4899", "Film"},
		{"cat_health", "Healthcare & Medical", "#14B8A6", "HeartPulse"},
		{"cat_salary", "Salary & Income", "#22C55E", "Wallet"},
		{"cat_transfers", "Self / P2P Transfers", "#8B5CF6", "ArrowLeftRight"},
		{"cat_charges", "Bank Charges & Interest", "#64748B", "Percent"},
		{"cat_others", "Others & Uncategorized", "#94A3B8", "HelpCircle"},
	}

	stmt, err := d.conn.Prepare(`
		INSERT INTO categories (id, name, color_hex, icon, is_system)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			color_hex = excluded.color_hex,
			icon = excluded.icon;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range defaultCategories {
		if _, err := stmt.Exec(c.ID, c.Name, c.ColorHex, c.Icon); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) seedDefaultRules() error {
	defaultRules := []struct {
		ID             string
		Priority       int
		Field          string
		Type           string
		Pattern        string
		ExcludePattern string
		TxType         string
		CatID          string
	}{
		{"rule_swiggy", 100, "cleaned_payee", "CONTAINS", "SWIGGY", "", "ALL", "cat_food"},
		{"rule_zomato", 100, "cleaned_payee", "CONTAINS", "ZOMATO", "", "ALL", "cat_food"},
		{"rule_blinkit", 95, "cleaned_payee", "CONTAINS", "BLINKIT", "", "ALL", "cat_groceries"},
		{"rule_zepto", 95, "cleaned_payee", "CONTAINS", "ZEPTO", "", "ALL", "cat_groceries"},
		{"rule_instamart", 95, "cleaned_payee", "CONTAINS", "INSTAMART", "", "ALL", "cat_groceries"},
		{"rule_amazon", 90, "cleaned_payee", "CONTAINS", "AMAZON", "AWS", "ALL", "cat_shopping"},
		{"rule_flipkart", 90, "cleaned_payee", "CONTAINS", "FLIPKART", "", "ALL", "cat_shopping"},
		{"rule_myntra", 90, "cleaned_payee", "CONTAINS", "MYNTRA", "", "ALL", "cat_shopping"},
		{"rule_uber", 85, "cleaned_payee", "CONTAINS", "UBER", "", "ALL", "cat_travel"},
		{"rule_ola", 85, "cleaned_payee", "CONTAINS", "OLA", "", "ALL", "cat_travel"},
		{"rule_petrol", 85, "raw_narration", "REGEX", "(?i)HPCL|BPCL|IOCL|PETROL|FUEL", "", "ALL", "cat_travel"},
		{"rule_netflix", 80, "cleaned_payee", "CONTAINS", "NETFLIX", "", "ALL", "cat_entertainment"},
		{"rule_spotify", 80, "cleaned_payee", "CONTAINS", "SPOTIFY", "", "ALL", "cat_entertainment"},
		{"rule_hotstar", 80, "cleaned_payee", "CONTAINS", "HOTSTAR", "", "ALL", "cat_entertainment"},
		{"rule_zerodha", 90, "raw_narration", "REGEX", "(?i)ZERODHA|GROWW|INDMONEY|KITE", "", "ALL", "cat_invest"},
		{"rule_salary", 100, "raw_narration", "REGEX", "(?i)SALARY|SAL CREDIT", "maid, driver, cook, helper, staff, advance", "CREDIT", "cat_salary"},
		{"rule_bank_charges", 90, "raw_narration", "REGEX", "(?i)CONSOLIDATED CHGS|SMS ALERT|ANNUAL FEE|GST", "", "ALL", "cat_charges"},
		{"rule_transfers", 100, "raw_narration", "REGEX", "(?i)IB FUNDS TRANSFER|TPT|INT-TRF|SELF TRANSFER|UPI-LITE|CRED.CLUB", "", "ALL", "cat_transfers"},
	}

	stmt, err := d.conn.Prepare(`
		INSERT INTO categorization_rules (id, priority, match_field, match_type, match_pattern, exclude_pattern, tx_type, target_category_id, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(id) DO NOTHING;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range defaultRules {
		if _, err := stmt.Exec(r.ID, r.Priority, r.Field, r.Type, r.Pattern, r.ExcludePattern, r.TxType, r.CatID); err != nil {
			return err
		}
	}
	return nil
}

// Account Operations
func (d *DB) ListAccounts() ([]models.Account, error) {
	rows, err := d.conn.Query(`
		SELECT id, bank_name, account_type, account_number, account_number_mask, currency, opening_balance, current_balance, credit_limit, billing_cycle_day, nickname, customer_id, ifsc_code, branch_name, card_network, card_variant, account_holder_name, created_at, updated_at
		FROM accounts
		ORDER BY bank_name ASC, account_type ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Account
	for rows.Next() {
		var a models.Account
		var accNum, accMask, nickname, custID, ifsc, branch, cardNet, cardVar, accHolder sql.NullString
		var credLimit sql.NullFloat64
		var billCycle sql.NullInt32
		if err := rows.Scan(&a.ID, &a.BankName, &a.AccountType, &accNum, &accMask, &a.Currency, &a.OpeningBalance, &a.CurrentBalance, &credLimit, &billCycle, &nickname, &custID, &ifsc, &branch, &cardNet, &cardVar, &accHolder, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if accNum.Valid && accNum.String != "" {
			a.AccountNumber = &accNum.String
		}
		if accMask.Valid {
			a.AccountNumberMask = accMask.String
		}
		if credLimit.Valid {
			val := credLimit.Float64
			a.CreditLimit = &val
		}
		if billCycle.Valid {
			val := int(billCycle.Int32)
			a.BillingCycleDay = &val
		}
		if nickname.Valid {
			a.Nickname = &nickname.String
		}
		if custID.Valid {
			a.CustomerID = &custID.String
		}
		if ifsc.Valid {
			a.IFSCCode = &ifsc.String
		}
		if branch.Valid {
			a.BranchName = &branch.String
		}
		if cardNet.Valid {
			a.CardNetwork = &cardNet.String
		}
		if cardVar.Valid {
			a.CardVariant = &cardVar.String
		}
		if accHolder.Valid {
			a.AccountHolderName = &accHolder.String
		}
		list = append(list, a)
	}
	return list, nil
}

func (d *DB) GetOrCreateAccount(bankName string, accType models.AccountType, accNum, mask string, custID, ifsc, branch, cardNet, cardVar, accHolder string, creditLimit *float64) (*models.Account, error) {
	var a models.Account
	var accNumber, accMask, nickname, customerID, ifscCode, branchName, cNet, cVar, holderName sql.NullString
	var credLimit sql.NullFloat64
	var billCycle sql.NullInt32

	var row *sql.Row
	if accNum != "" {
		// Full numbers override mask similarity. A mask-only account can be
		// upgraded when its labeled mask matches, but different known full
		// account numbers must never merge solely because their last four match.
		row = d.conn.QueryRow(`
			SELECT id, bank_name, account_type, account_number, account_number_mask, currency, opening_balance, current_balance, credit_limit, billing_cycle_day, nickname, customer_id, ifsc_code, branch_name, card_network, card_variant, account_holder_name, created_at, updated_at
			FROM accounts
			WHERE bank_name = ? AND (account_number = ? OR
				(account_type = ? AND ? <> '' AND account_number_mask = ? AND (account_number IS NULL OR account_number = '')))
			ORDER BY CASE WHEN account_number = ? THEN 0 ELSE 1 END
			LIMIT 1
		`, bankName, accNum, accType, mask, mask, accNum)
	} else if mask != "" {
		// Match by bank, account_type and mask
		row = d.conn.QueryRow(`
			SELECT id, bank_name, account_type, account_number, account_number_mask, currency, opening_balance, current_balance, credit_limit, billing_cycle_day, nickname, customer_id, ifsc_code, branch_name, card_network, card_variant, account_holder_name, created_at, updated_at
			FROM accounts
			WHERE bank_name = ? AND account_type = ? AND account_number_mask = ?
			LIMIT 1
		`, bankName, accType, mask)
	} else {
		// Match only an account with no account number and no mask
		row = d.conn.QueryRow(`
			SELECT id, bank_name, account_type, account_number, account_number_mask, currency, opening_balance, current_balance, credit_limit, billing_cycle_day, nickname, customer_id, ifsc_code, branch_name, card_network, card_variant, account_holder_name, created_at, updated_at
			FROM accounts
			WHERE bank_name = ? AND account_type = ? AND (account_number IS NULL OR account_number = '') AND (account_number_mask IS NULL OR account_number_mask = '')
			LIMIT 1
		`, bankName, accType)
	}

	err := row.Scan(&a.ID, &a.BankName, &a.AccountType, &accNumber, &accMask, &a.Currency, &a.OpeningBalance, &a.CurrentBalance, &credLimit, &billCycle, &nickname, &customerID, &ifscCode, &branchName, &cNet, &cVar, &holderName, &a.CreatedAt, &a.UpdatedAt)

	if err == nil {
		if accNumber.Valid && accNumber.String != "" {
			a.AccountNumber = &accNumber.String
		}
		if accMask.Valid {
			a.AccountNumberMask = accMask.String
		}
		if credLimit.Valid {
			val := credLimit.Float64
			a.CreditLimit = &val
		}
		if billCycle.Valid {
			val := int(billCycle.Int32)
			a.BillingCycleDay = &val
		}
		if nickname.Valid {
			a.Nickname = &nickname.String
		}
		if customerID.Valid {
			a.CustomerID = &customerID.String
		}
		if ifscCode.Valid {
			a.IFSCCode = &ifscCode.String
		}
		if branchName.Valid {
			a.BranchName = &branchName.String
		}
		if cNet.Valid {
			a.CardNetwork = &cNet.String
		}
		if cVar.Valid {
			a.CardVariant = &cVar.String
		}
		if holderName.Valid {
			a.AccountHolderName = &holderName.String
		}
		// Update identifiers if previously empty
		if accNum != "" && (a.AccountNumber == nil || *a.AccountNumber == "") {
			_, _ = d.conn.Exec(`UPDATE accounts SET account_number = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, accNum, a.ID)
			a.AccountNumber = &accNum
		}
		if mask != "" && a.AccountNumberMask == "" {
			_, _ = d.conn.Exec(`UPDATE accounts SET account_number_mask = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, mask, a.ID)
			a.AccountNumberMask = mask
		}
		if cardNet != "" && a.CardNetwork == nil {
			_, _ = d.conn.Exec(`UPDATE accounts SET card_network = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, cardNet, a.ID)
			a.CardNetwork = &cardNet
		}
		if cardVar != "" && a.CardVariant == nil {
			_, _ = d.conn.Exec(`UPDATE accounts SET card_variant = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, cardVar, a.ID)
			a.CardVariant = &cardVar
		}
		if accHolder != "" && (a.AccountHolderName == nil || *a.AccountHolderName == "") {
			_, _ = d.conn.Exec(`UPDATE accounts SET account_holder_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, accHolder, a.ID)
			a.AccountHolderName = &accHolder
		}
		if creditLimit != nil && a.CreditLimit == nil {
			_, _ = d.conn.Exec(`UPDATE accounts SET credit_limit = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *creditLimit, a.ID)
			a.CreditLimit = creditLimit
		}
		return &a, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new account
	newID := uuid.New().String()
	now := time.Now()
	_, err = d.conn.Exec(`
		INSERT INTO accounts (id, bank_name, account_type, account_number, account_number_mask, currency, opening_balance, current_balance, credit_limit, customer_id, ifsc_code, branch_name, card_network, card_variant, account_holder_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'INR', 0, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, newID, bankName, accType, accNum, mask, creditLimit, custID, ifsc, branch, cardNet, cardVar, accHolder, now, now)
	if err != nil {
		return nil, err
	}

	res := &models.Account{
		ID:                newID,
		BankName:          bankName,
		AccountType:       accType,
		AccountNumberMask: mask,
		Currency:          "INR",
		OpeningBalance:    0,
		CurrentBalance:    0,
		CreditLimit:       creditLimit,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if accNum != "" {
		res.AccountNumber = &accNum
	}
	if custID != "" {
		res.CustomerID = &custID
	}
	if ifsc != "" {
		res.IFSCCode = &ifsc
	}
	if branch != "" {
		res.BranchName = &branch
	}
	if cardNet != "" {
		res.CardNetwork = &cardNet
	}
	if cardVar != "" {
		res.CardVariant = &cardVar
	}
	if accHolder != "" {
		res.AccountHolderName = &accHolder
	}

	if accType == models.AccountTypeCreditCard {
		d.SeedDefaultCardRulesIfEmpty(res.ID, cardVar, bankName)
	}

	return res, nil
}

func (d *DB) CreateStatementImport(s *models.StatementImport) error {
	_, err := d.conn.Exec(`
		INSERT INTO statement_imports (
			id, account_id, filename, file_hash, statement_format, parser_used,
			start_date, end_date, total_transactions, opening_balance, closing_balance, total_debits, total_credits, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.AccountID, s.Filename, s.FileHash, s.StatementFormat, s.ParserUsed,
		s.StartDate, s.EndDate, s.TotalTransactions, s.OpeningBalance, s.ClosingBalance, s.TotalDebits, s.TotalCredits, s.ImportedAt)
	return err
}

func (d *DB) ListStatementImports() ([]models.StatementImport, error) {
	rows, err := d.conn.Query(`
		SELECT s.id, s.account_id, a.bank_name || ' (' || a.account_type || ')' as bank_name,
		       s.filename, s.file_hash, s.statement_format, s.parser_used, s.start_date, s.end_date, s.total_transactions,
		       s.opening_balance, s.closing_balance, s.total_debits, s.total_credits, s.imported_at
		FROM statement_imports s
		JOIN accounts a ON s.account_id = a.id
		ORDER BY s.imported_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.StatementImport
	for rows.Next() {
		var s models.StatementImport
		var openBal, closeBal, totDeb, totCred sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.AccountID, &s.BankName, &s.Filename, &s.FileHash, &s.StatementFormat, &s.ParserUsed,
			&s.StartDate, &s.EndDate, &s.TotalTransactions, &openBal, &closeBal, &totDeb, &totCred, &s.ImportedAt); err != nil {
			return nil, err
		}
		if openBal.Valid {
			s.OpeningBalance = &openBal.Float64
		}
		if closeBal.Valid {
			s.ClosingBalance = &closeBal.Float64
		}
		if totDeb.Valid {
			s.TotalDebits = &totDeb.Float64
		}
		if totCred.Valid {
			s.TotalCredits = &totCred.Float64
		}
		list = append(list, s)
	}
	return list, nil
}

// Transaction Upsert
func (d *DB) UpsertTransaction(tx *models.Transaction) (bool, error) {
	var existingID string
	var existingCatID sql.NullString
	var existingNotes sql.NullString
	var existingTags sql.NullString
	var existingManualCat sql.NullBool

	err := d.conn.QueryRow(`
		SELECT id, category_id, notes, tags, is_manual_category FROM transactions WHERE tx_hash = ?
	`, tx.TxHash).Scan(&existingID, &existingCatID, &existingNotes, &existingTags, &existingManualCat)

	isNew := false
	if err == sql.ErrNoRows {
		isNew = true
		tx.ID = uuid.New().String()
		tx.CreatedAt = time.Now()

		isManual := 0
		if tx.IsManualCategory {
			isManual = 1
		}

		_, err := d.conn.Exec(`
			INSERT INTO transactions (
				id, account_id, statement_import_id, tx_hash, tx_date, value_date,
				raw_narration, cleaned_payee, payment_mode, reference_number,
				tx_type, amount, running_balance, category_id, is_recurring, is_manual_category, notes, tags,
				upi_vpa, card_last4, merchant_category, cashback_amount, reward_points_earned,
				is_transfer, is_excluded, original_currency, original_amount, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			tx.ID, tx.AccountID, tx.StatementImportID, tx.TxHash, tx.TxDate, tx.ValueDate,
			tx.RawNarration, tx.CleanedPayee, tx.PaymentMode, tx.ReferenceNumber,
			tx.TxType, tx.Amount, tx.RunningBalance, tx.CategoryID, tx.IsRecurring, isManual, tx.Notes, tx.Tags,
			tx.UPIVPA, tx.CardLast4, tx.MerchantCategory, tx.CashbackAmount, tx.RewardPointsEarned,
			tx.IsTransfer, tx.IsExcluded, tx.OriginalCurrency, tx.OriginalAmount, tx.CreatedAt,
		)
		return isNew, err
	} else if err != nil {
		return false, err
	}

	// Update metadata while strictly preserving user categorized fields
	tx.ID = existingID
	var catIDToKeep *string = tx.CategoryID
	if existingCatID.Valid && existingCatID.String != "" {
		val := existingCatID.String
		catIDToKeep = &val
	}
	notesToKeep := tx.Notes
	if existingNotes.Valid && existingNotes.String != "" {
		notesToKeep = existingNotes.String
	}
	tagsToKeep := tx.Tags
	if existingTags.Valid && existingTags.String != "" {
		tagsToKeep = existingTags.String
	}
	manualCatToKeep := 0
	if existingManualCat.Valid && existingManualCat.Bool {
		manualCatToKeep = 1
	}

	_, err = d.conn.Exec(`
		UPDATE transactions SET
			statement_import_id = COALESCE(?, statement_import_id),
			value_date = COALESCE(?, value_date),
			cleaned_payee = COALESCE(?, cleaned_payee),
			payment_mode = COALESCE(?, payment_mode),
			reference_number = COALESCE(?, reference_number),
			running_balance = COALESCE(?, running_balance),
			upi_vpa = COALESCE(?, upi_vpa),
			card_last4 = COALESCE(?, card_last4),
			merchant_category = COALESCE(?, merchant_category),
			cashback_amount = CASE WHEN ? > 0 THEN ? ELSE cashback_amount END,
			reward_points_earned = CASE WHEN ? > 0 THEN ? ELSE reward_points_earned END,
			is_transfer = CASE WHEN transfer_peer_id IS NOT NULL OR is_transfer = 1 THEN 1 ELSE ? END,
			category_id = ?,
			is_manual_category = ?,
			notes = ?,
			tags = ?
		WHERE tx_hash = ?
	`, tx.StatementImportID, tx.ValueDate, tx.CleanedPayee, tx.PaymentMode, tx.ReferenceNumber,
		tx.RunningBalance, tx.UPIVPA, tx.CardLast4, tx.MerchantCategory, tx.CashbackAmount, tx.CashbackAmount,
		tx.RewardPointsEarned, tx.RewardPointsEarned, tx.IsTransfer, catIDToKeep, manualCatToKeep, notesToKeep, tagsToKeep, tx.TxHash)

	return isNew, err
}

func (d *DB) CheckTxHashExists(txHash string) (bool, error) {
	var count int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM transactions WHERE tx_hash = ?", txHash).Scan(&count)
	return count > 0, err
}

func (d *DB) RecalculateAccountBalance(accountID string) error {
	var latestBal sql.NullFloat64
	err := d.conn.QueryRow(`
		SELECT running_balance FROM transactions
		WHERE account_id = ? AND running_balance IS NOT NULL
		ORDER BY tx_date DESC, created_at DESC
		LIMIT 1
	`, accountID).Scan(&latestBal)

	if err == nil && latestBal.Valid {
		_, err := d.conn.Exec(`UPDATE accounts SET current_balance = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, latestBal.Float64, accountID)
		return err
	}

	var netChange sql.NullFloat64
	err = d.conn.QueryRow(`
		SELECT SUM(CASE WHEN tx_type = 'CREDIT' THEN amount ELSE -amount END)
		FROM transactions WHERE account_id = ?
	`, accountID).Scan(&netChange)
	if err != nil {
		return err
	}

	bal := 0.0
	if netChange.Valid {
		bal = netChange.Float64
	}

	_, err = d.conn.Exec(`UPDATE accounts SET current_balance = opening_balance + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, bal, accountID)
	return err
}

type TransactionFilter struct {
	ID         string
	AccountID  string
	CategoryID string
	TxType     string
	Search     string
	StartDate  string
	EndDate    string
	IsTransfer *bool
	Limit      int
	Offset     int
}

func (d *DB) ListTransactions(f TransactionFilter) ([]models.Transaction, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if f.ID != "" {
		where = append(where, "t.id = ?")
		args = append(args, f.ID)
	}
	if f.AccountID != "" {
		where = append(where, "t.account_id = ?")
		args = append(args, f.AccountID)
	}
	if f.CategoryID != "" {
		where = append(where, "t.category_id = ?")
		args = append(args, f.CategoryID)
	}
	if f.TxType != "" {
		where = append(where, "t.tx_type = ?")
		args = append(args, f.TxType)
	}
	if f.StartDate != "" {
		where = append(where, "t.tx_date >= ?")
		args = append(args, f.StartDate)
	}
	if f.EndDate != "" {
		where = append(where, "t.tx_date <= ?")
		args = append(args, f.EndDate)
	}
	if f.IsTransfer != nil {
		where = append(where, "t.is_transfer = ?")
		args = append(args, *f.IsTransfer)
	}
	if f.Search != "" {
		where = append(where, "(t.raw_narration LIKE ? OR t.cleaned_payee LIKE ? OR t.reference_number LIKE ? OR t.upi_vpa LIKE ?)")
		searchTerm := "%" + f.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transactions t WHERE %s", whereClause)
	if err := d.conn.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT 
			t.id, t.account_id, a.bank_name || ' (' || a.account_type || ')' as account_name,
			t.statement_import_id, t.tx_hash, t.tx_date, t.value_date,
			t.raw_narration, t.cleaned_payee, t.payment_mode, t.reference_number,
			t.tx_type, t.amount, t.running_balance,
			t.category_id, c.name as category_name, c.color_hex as category_color, c.icon as category_icon,
			t.upi_vpa, t.card_last4, t.merchant_category, t.cashback_amount, t.reward_points_earned,
			t.is_transfer, t.is_excluded, t.transfer_peer_id, t.transfer_match_reason, t.net_amount,
			t.original_currency, t.original_amount,
			t.is_recurring, COALESCE(t.is_manual_category, 0), COALESCE(t.notes, ''), COALESCE(t.tags, ''), t.created_at
		FROM transactions t
		LEFT JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE %s
		ORDER BY t.tx_date DESC, t.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := d.conn.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var stmtID, valDate, catID, catName, catColor, catIcon, upiVpa, cardLast4, merchCat, peerID, matchReason, origCurr sql.NullString
		var runBal, netAmt, origAmt sql.NullFloat64
		var isManualCat sql.NullBool

		if err := rows.Scan(
			&t.ID, &t.AccountID, &t.AccountName,
			&stmtID, &t.TxHash, &t.TxDate, &valDate,
			&t.RawNarration, &t.CleanedPayee, &t.PaymentMode, &t.ReferenceNumber,
			&t.TxType, &t.Amount, &runBal,
			&catID, &catName, &catColor, &catIcon,
			&upiVpa, &cardLast4, &merchCat, &t.CashbackAmount, &t.RewardPointsEarned,
			&t.IsTransfer, &t.IsExcluded, &peerID, &matchReason, &netAmt,
			&origCurr, &origAmt,
			&t.IsRecurring, &isManualCat, &t.Notes, &t.Tags, &t.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		if stmtID.Valid {
			t.StatementImportID = &stmtID.String
		}
		if valDate.Valid {
			t.ValueDate = &valDate.String
		}
		if runBal.Valid {
			val := runBal.Float64
			t.RunningBalance = &val
		}
		if catID.Valid {
			t.CategoryID = &catID.String
		}
		if catName.Valid {
			t.CategoryName = &catName.String
		}
		if catColor.Valid {
			t.CategoryColor = &catColor.String
		}
		if catIcon.Valid {
			t.CategoryIcon = &catIcon.String
		}
		if upiVpa.Valid {
			t.UPIVPA = &upiVpa.String
		}
		if cardLast4.Valid {
			t.CardLast4 = &cardLast4.String
		}
		if merchCat.Valid {
			t.MerchantCategory = &merchCat.String
		}
		if peerID.Valid {
			t.TransferPeerID = &peerID.String
		}
		if matchReason.Valid {
			t.TransferMatchReason = &matchReason.String
		}
		if netAmt.Valid {
			val := netAmt.Float64
			t.NetAmount = &val
		}
		if origCurr.Valid {
			t.OriginalCurrency = &origCurr.String
		}
		if origAmt.Valid {
			val := origAmt.Float64
			t.OriginalAmount = &val
		}
		if isManualCat.Valid {
			t.IsManualCategory = isManualCat.Bool
		}

		list = append(list, t)
	}

	return list, total, nil
}

func (d *DB) GetTransaction(id string) (*models.Transaction, error) {
	list, total, err := d.ListTransactions(TransactionFilter{ID: id, Limit: 1})
	if err != nil {
		return nil, err
	}
	if total == 0 || len(list) == 0 {
		return nil, sql.ErrNoRows
	}
	return &list[0], nil
}

func (d *DB) UpdateTransaction(id string, req models.UpdateTransactionRequest) (*models.Transaction, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Verify transaction exists
	var currentCatID sql.NullString
	err := d.conn.QueryRow(`SELECT category_id FROM transactions WHERE id = ?`, id).Scan(&currentCatID)
	if err != nil {
		return nil, err
	}

	sets := []string{}
	args := []interface{}{}

	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			sets = append(sets, "category_id = NULL")
		} else {
			var catCount int
			_ = d.conn.QueryRow(`SELECT COUNT(*) FROM categories WHERE id = ?`, *req.CategoryID).Scan(&catCount)
			if catCount == 0 {
				return nil, fmt.Errorf("category '%s' not found", *req.CategoryID)
			}
			sets = append(sets, "category_id = ?")
			args = append(args, *req.CategoryID)
		}

		isManual := 1
		if req.IsManualCategory != nil && !*req.IsManualCategory {
			isManual = 0
		}
		sets = append(sets, "is_manual_category = ?")
		args = append(args, isManual)
	} else if req.IsManualCategory != nil {
		isManual := 0
		if *req.IsManualCategory {
			isManual = 1
		}
		sets = append(sets, "is_manual_category = ?")
		args = append(args, isManual)
	}

	if req.Notes != nil {
		sets = append(sets, "notes = ?")
		args = append(args, *req.Notes)
	}

	if req.Tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, *req.Tags)
	}

	if len(sets) > 0 {
		query := fmt.Sprintf(`UPDATE transactions SET %s WHERE id = ?`, strings.Join(sets, ", "))
		args = append(args, id)
		if _, err := d.conn.Exec(query, args...); err != nil {
			return nil, err
		}
	}

	// Fetch updated transaction
	list, total, err := d.ListTransactions(TransactionFilter{ID: id, Limit: 1})
	if err != nil {
		return nil, err
	}
	if total == 0 || len(list) == 0 {
		return nil, sql.ErrNoRows
	}
	return &list[0], nil
}

func (d *DB) ListCategories() ([]models.Category, error) {
	rows, err := d.conn.Query(`
		SELECT id, name, parent_id, color_hex, icon, is_system
		FROM categories
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Category
	for rows.Next() {
		var c models.Category
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &parentID, &c.ColorHex, &c.Icon, &c.IsSystem); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = &parentID.String
		}
		list = append(list, c)
	}
	return list, nil
}

func (d *DB) ListRules() ([]models.CategorizationRule, error) {
	rows, err := d.conn.Query(`
		SELECT r.id, r.priority, r.match_field, r.match_type, r.match_pattern, COALESCE(r.exclude_pattern, ''), COALESCE(r.tx_type, 'ALL'), r.target_category_id, c.name, COALESCE(r.assign_tags, ''), r.is_active
		FROM categorization_rules r
		LEFT JOIN categories c ON r.target_category_id = c.id
		WHERE r.is_active = 1
		ORDER BY r.priority DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CategorizationRule
	for rows.Next() {
		var r models.CategorizationRule
		var catName sql.NullString
		if err := rows.Scan(&r.ID, &r.Priority, &r.MatchField, &r.MatchType, &r.MatchPattern, &r.ExcludePattern, &r.TxType, &r.TargetCategoryID, &catName, &r.AssignTags, &r.IsActive); err != nil {
			return nil, err
		}
		if catName.Valid {
			r.TargetCategory = catName.String
		}
		list = append(list, r)
	}
	return list, nil
}

func (d *DB) CreateRule(r *models.CategorizationRule) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.Priority == 0 {
		r.Priority = 50
	}
	if r.TxType == "" {
		r.TxType = "ALL"
	}

	_, err := d.conn.Exec(`
		INSERT INTO categorization_rules (id, priority, match_field, match_type, match_pattern, exclude_pattern, tx_type, target_category_id, assign_tags, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.Priority, r.MatchField, r.MatchType, r.MatchPattern, r.ExcludePattern, r.TxType, r.TargetCategoryID, r.AssignTags, r.IsActive)
	return err
}

func (d *DB) UpdateRule(r *models.CategorizationRule) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if r.TxType == "" {
		r.TxType = "ALL"
	}

	_, err := d.conn.Exec(`
		UPDATE categorization_rules SET
			priority = ?, match_field = ?, match_type = ?, match_pattern = ?,
			exclude_pattern = ?, tx_type = ?,
			target_category_id = ?, assign_tags = ?, is_active = ?
		WHERE id = ?
	`, r.Priority, r.MatchField, r.MatchType, r.MatchPattern, r.ExcludePattern, r.TxType, r.TargetCategoryID, r.AssignTags, r.IsActive, r.ID)
	return err
}

func (d *DB) DeleteRule(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec(`DELETE FROM categorization_rules WHERE id = ?`, id)
	return err
}

func (d *DB) CreateCategory(c *models.Category) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.ColorHex == "" {
		c.ColorHex = "#64748B"
	}
	if c.Icon == "" {
		c.Icon = "tag"
	}

	_, err := d.conn.Exec(`
		INSERT INTO categories (id, name, parent_id, color_hex, icon, is_system)
		VALUES (?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.ParentID, c.ColorHex, c.Icon, c.IsSystem)
	return err
}

func (d *DB) DeleteCategory(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update transactions referencing this category to NULL
	_, _ = d.conn.Exec(`UPDATE transactions SET category_id = NULL WHERE category_id = ?`, id)
	// Delete any rules targeting this category
	_, _ = d.conn.Exec(`DELETE FROM categorization_rules WHERE target_category_id = ?`, id)
	// Delete category
	_, err := d.conn.Exec(`DELETE FROM categories WHERE id = ? AND is_system = 0`, id)
	return err
}

func (d *DB) ReapplyRules() (int, error) {
	rules, err := d.ListRules()
	if err != nil {
		return 0, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.conn.Query(`
		SELECT id, raw_narration, cleaned_payee, reference_number, upi_vpa, category_id, tx_type
		FROM transactions
		WHERE COALESCE(is_manual_category, 0) = 0
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type txItem struct {
		id         string
		narration  string
		payee      string
		ref        string
		vpa        string
		currentCat *string
		txType     string
	}
	var txs []txItem
	for rows.Next() {
		var item txItem
		var vpa, cat sql.NullString
		if err := rows.Scan(&item.id, &item.narration, &item.payee, &item.ref, &vpa, &cat, &item.txType); err != nil {
			return 0, err
		}
		if vpa.Valid {
			item.vpa = vpa.String
		}
		if cat.Valid {
			item.currentCat = &cat.String
		}
		txs = append(txs, item)
	}

	updatedCount := 0
	txStmt, err := d.conn.Prepare(`UPDATE transactions SET category_id = ? WHERE id = ?`)
	if err != nil {
		return 0, err
	}
	defer txStmt.Close()

	for _, item := range txs {
		var matchedCatID *string
		for _, r := range rules {
			if !r.MatchesTxType(item.txType) {
				continue
			}
			if r.MatchesException(item.narration, item.payee) {
				continue
			}

			targetVal := ""
			switch r.MatchField {
			case "cleaned_payee":
				targetVal = item.payee
			case "raw_narration":
				targetVal = item.narration
			case "reference_number":
				targetVal = item.ref
			case "upi_vpa":
				targetVal = item.vpa
			default:
				targetVal = item.narration
			}

			matched := false
			switch r.MatchType {
			case "CONTAINS":
				matched = strings.Contains(strings.ToUpper(targetVal), strings.ToUpper(r.MatchPattern))
			case "EXACT":
				matched = strings.EqualFold(targetVal, r.MatchPattern)
			case "STARTS_WITH":
				matched = strings.HasPrefix(strings.ToUpper(targetVal), strings.ToUpper(r.MatchPattern))
			case "REGEX":
				if re, err := regexp.Compile(r.MatchPattern); err == nil {
					matched = re.MatchString(targetVal)
				}
			}

			if matched {
				cid := r.TargetCategoryID
				matchedCatID = &cid
				break
			}
		}

		if matchedCatID != nil {
			if item.currentCat == nil || *item.currentCat != *matchedCatID {
				if _, err := txStmt.Exec(*matchedCatID, item.id); err == nil {
					updatedCount++
				}
			}
		} else {
			// If no rule matches, but transaction was previously categorized as "cat_salary":
			// Salary & Income is strictly an income category (CREDIT).
			// If it's a DEBIT, or if it matched an exclusion keyword (e.g. cook, maid, staff),
			// or if no active salary rule matches it, reset it to "cat_others".
			if item.currentCat != nil && *item.currentCat == "cat_salary" {
				catOthers := "cat_others"
				if _, err := txStmt.Exec(catOthers, item.id); err == nil {
					updatedCount++
				}
			}
		}
	}

	return updatedCount, nil
}

func (d *DB) CreateOrUpdateCreditCardBill(b *models.CreditCardBill) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	b.CreatedAt = time.Now()

	_, err := d.conn.Exec(`
		INSERT INTO credit_card_bills (
			id, account_id, statement_import_id, statement_date, payment_due_date,
			total_due_amount, minimum_due_amount, reward_points_earned, reward_points_balance,
			cashback_earned, cashback_credited, finance_charges, credit_limit, available_credit_limit,
			payment_status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			total_due_amount = excluded.total_due_amount,
			minimum_due_amount = excluded.minimum_due_amount,
			reward_points_earned = excluded.reward_points_earned,
			reward_points_balance = excluded.reward_points_balance,
			cashback_earned = excluded.cashback_earned,
			cashback_credited = excluded.cashback_credited,
			finance_charges = excluded.finance_charges,
			credit_limit = excluded.credit_limit,
			available_credit_limit = excluded.available_credit_limit,
			payment_status = excluded.payment_status;
	`, b.ID, b.AccountID, b.StatementImportID, b.StatementDate, b.PaymentDueDate,
		b.TotalDueAmount, b.MinimumDueAmount, b.RewardPointsEarned, b.RewardPointsBalance,
		b.CashbackEarned, b.CashbackCredited, b.FinanceCharges, b.CreditLimit, b.AvailableCreditLimit,
		b.PaymentStatus, b.CreatedAt)
	return err
}

func (d *DB) ListCreditCardBills(accountID string) ([]models.CreditCardBill, error) {
	where := "1=1"
	args := []interface{}{}
	if accountID != "" {
		where = "b.account_id = ?"
		args = append(args, accountID)
	}

	query := fmt.Sprintf(`
		SELECT b.id, b.account_id, a.bank_name || ' (' || a.account_type || ')' as bank_name,
		       b.statement_import_id, b.statement_date, b.payment_due_date,
		       b.total_due_amount, b.minimum_due_amount, b.reward_points_earned, b.reward_points_balance,
		       b.cashback_earned, b.cashback_credited, b.finance_charges, b.credit_limit, b.available_credit_limit,
		       b.payment_status, b.created_at
		FROM credit_card_bills b
		JOIN accounts a ON b.account_id = a.id
		WHERE %s
		ORDER BY b.statement_date DESC
	`, where)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CreditCardBill
	for rows.Next() {
		var b models.CreditCardBill
		var stmtID sql.NullString
		var minDue, credLim, availLim sql.NullFloat64
		if err := rows.Scan(&b.ID, &b.AccountID, &b.BankName, &stmtID, &b.StatementDate, &b.PaymentDueDate,
			&b.TotalDueAmount, &minDue, &b.RewardPointsEarned, &b.RewardPointsBalance,
			&b.CashbackEarned, &b.CashbackCredited, &b.FinanceCharges, &credLim, &availLim,
			&b.PaymentStatus, &b.CreatedAt); err != nil {
			return nil, err
		}
		if stmtID.Valid {
			b.StatementImportID = &stmtID.String
		}
		if minDue.Valid {
			val := minDue.Float64
			b.MinimumDueAmount = &val
		}
		if credLim.Valid {
			val := credLim.Float64
			b.CreditLimit = &val
		}
		if availLim.Valid {
			val := availLim.Float64
			b.AvailableCreditLimit = &val
		}
		list = append(list, b)
	}
	return list, nil
}

func (d *DB) ListSubscriptions() ([]models.Subscription, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT 
			s.id, s.name, s.merchant_pattern, s.category_id, c.name, c.color_hex, c.icon,
			s.account_id, a.bank_name || ' (' || a.account_type || CASE WHEN a.account_number_mask IS NOT NULL AND a.account_number_mask != '' THEN ' ' || a.account_number_mask ELSE '' END || ')',
			s.frequency, s.expected_amount, s.currency, s.billing_day,
			s.next_due_date, s.last_paid_date, s.last_paid_amount, s.status, s.is_auto_detected, s.notes,
			s.created_at, s.updated_at
		FROM subscriptions s
		LEFT JOIN categories c ON s.category_id = c.id
		LEFT JOIN accounts a ON s.account_id = a.id
		ORDER BY 
			CASE s.status WHEN 'ACTIVE' THEN 1 WHEN 'PAUSED' THEN 2 ELSE 3 END,
			s.next_due_date ASC, s.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Subscription
	for rows.Next() {
		var s models.Subscription
		var catID, catName, catColor, catIcon, accID, accName, nextDue, lastPaid, notes sql.NullString
		var billingDay sql.NullInt64
		var lastPaidAmt sql.NullFloat64
		var isAuto int

		err := rows.Scan(
			&s.ID, &s.Name, &s.MerchantPattern, &catID, &catName, &catColor, &catIcon,
			&accID, &accName,
			&s.Frequency, &s.ExpectedAmount, &s.Currency, &billingDay,
			&nextDue, &lastPaid, &lastPaidAmt, &s.Status, &isAuto, &notes,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if catID.Valid {
			s.CategoryID = &catID.String
		}
		if catName.Valid {
			s.CategoryName = &catName.String
		}
		if catColor.Valid {
			s.CategoryColor = &catColor.String
		}
		if catIcon.Valid {
			s.CategoryIcon = &catIcon.String
		}
		if accID.Valid {
			s.AccountID = &accID.String
		}
		if accName.Valid {
			s.AccountName = &accName.String
		}
		if billingDay.Valid {
			s.BillingDay = int(billingDay.Int64)
		}
		if nextDue.Valid {
			s.NextDueDate = &nextDue.String
		}
		if lastPaid.Valid {
			s.LastPaidDate = &lastPaid.String
		}
		if lastPaidAmt.Valid {
			s.LastPaidAmount = &lastPaidAmt.Float64
		}
		if notes.Valid {
			s.Notes = &notes.String
		}
		s.IsAutoDetected = isAuto == 1

		list = append(list, s)
	}
	return list, nil
}

func (d *DB) GetSubscription(id string) (*models.Subscription, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var s models.Subscription
	var catID, catName, catColor, catIcon, accID, accName, nextDue, lastPaid, notes sql.NullString
	var billingDay sql.NullInt64
	var lastPaidAmt sql.NullFloat64
	var isAuto int

	err := d.conn.QueryRow(`
		SELECT 
			s.id, s.name, s.merchant_pattern, s.category_id, c.name, c.color_hex, c.icon,
			s.account_id, a.bank_name || ' (' || a.account_type || CASE WHEN a.account_number_mask IS NOT NULL AND a.account_number_mask != '' THEN ' ' || a.account_number_mask ELSE '' END || ')',
			s.frequency, s.expected_amount, s.currency, s.billing_day,
			s.next_due_date, s.last_paid_date, s.last_paid_amount, s.status, s.is_auto_detected, s.notes,
			s.created_at, s.updated_at
		FROM subscriptions s
		LEFT JOIN categories c ON s.category_id = c.id
		LEFT JOIN accounts a ON s.account_id = a.id
		WHERE s.id = ?
	`, id).Scan(
		&s.ID, &s.Name, &s.MerchantPattern, &catID, &catName, &catColor, &catIcon,
		&accID, &accName,
		&s.Frequency, &s.ExpectedAmount, &s.Currency, &billingDay,
		&nextDue, &lastPaid, &lastPaidAmt, &s.Status, &isAuto, &notes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if catID.Valid {
		s.CategoryID = &catID.String
	}
	if catName.Valid {
		s.CategoryName = &catName.String
	}
	if catColor.Valid {
		s.CategoryColor = &catColor.String
	}
	if catIcon.Valid {
		s.CategoryIcon = &catIcon.String
	}
	if accID.Valid {
		s.AccountID = &accID.String
	}
	if accName.Valid {
		s.AccountName = &accName.String
	}
	if billingDay.Valid {
		s.BillingDay = int(billingDay.Int64)
	}
	if nextDue.Valid {
		s.NextDueDate = &nextDue.String
	}
	if lastPaid.Valid {
		s.LastPaidDate = &lastPaid.String
	}
	if lastPaidAmt.Valid {
		s.LastPaidAmount = &lastPaidAmt.Float64
	}
	if notes.Valid {
		s.Notes = &notes.String
	}
	s.IsAutoDetected = isAuto == 1

	return &s, nil
}

func (d *DB) CreateSubscription(s *models.Subscription) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	isAuto := 0
	if s.IsAutoDetected {
		isAuto = 1
	}

	_, err := d.conn.Exec(`
		INSERT INTO subscriptions (
			id, name, merchant_pattern, category_id, account_id,
			frequency, expected_amount, currency, billing_day,
			next_due_date, last_paid_date, last_paid_amount, status,
			is_auto_detected, notes, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		s.ID, s.Name, s.MerchantPattern, s.CategoryID, s.AccountID,
		s.Frequency, s.ExpectedAmount, s.Currency, s.BillingDay,
		s.NextDueDate, s.LastPaidDate, s.LastPaidAmount, s.Status,
		isAuto, s.Notes, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (d *DB) UpdateSubscription(s *models.Subscription) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	s.UpdatedAt = time.Now()
	isAuto := 0
	if s.IsAutoDetected {
		isAuto = 1
	}

	_, err := d.conn.Exec(`
		UPDATE subscriptions SET
			name = ?, merchant_pattern = ?, category_id = ?, account_id = ?,
			frequency = ?, expected_amount = ?, currency = ?, billing_day = ?,
			next_due_date = ?, last_paid_date = ?, last_paid_amount = ?, status = ?,
			is_auto_detected = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`,
		s.Name, s.MerchantPattern, s.CategoryID, s.AccountID,
		s.Frequency, s.ExpectedAmount, s.Currency, s.BillingDay,
		s.NextDueDate, s.LastPaidDate, s.LastPaidAmount, s.Status,
		isAuto, s.Notes, s.UpdatedAt, s.ID,
	)
	return err
}

func (d *DB) DeleteSubscription(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec("DELETE FROM subscriptions WHERE id = ?", id)
	return err
}

func (d *DB) MarkTransactionsRecurring(txIDs []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(txIDs) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE transactions SET is_recurring = 1 WHERE id IN ('%s')", strings.Join(txIDs, "','"))
	_, err := d.conn.Exec(query)
	return err
}

func (d *DB) GetSubscriptionsSummary() (*models.SubscriptionsSummary, error) {
	subs, err := d.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	if subs == nil {
		subs = []models.Subscription{}
	}

	summary := &models.SubscriptionsSummary{
		Subscriptions: subs,
	}

	for _, s := range subs {
		if s.Status == models.SubscriptionStatusActive {
			summary.TotalActive++
			var monthlyEquiv float64
			switch s.Frequency {
			case models.FrequencyMonthly:
				monthlyEquiv = s.ExpectedAmount
			case models.FrequencyQuarterly:
				monthlyEquiv = s.ExpectedAmount / 3.0
			case models.FrequencyYearly:
				monthlyEquiv = s.ExpectedAmount / 12.0
			case models.FrequencyWeekly:
				monthlyEquiv = s.ExpectedAmount * 4.33
			default:
				monthlyEquiv = s.ExpectedAmount
			}
			summary.MonthlyBurnRate += monthlyEquiv
			summary.AnnualProjected += monthlyEquiv * 12.0

			if s.NextDueDate != nil && *s.NextDueDate != "" {
				summary.UpcomingIn30Days++
			}
		}
	}

	return summary, nil
}

func (d *DB) GetAnalyticsOverview() (*models.AnalyticsOverview, error) {
	overview := &models.AnalyticsOverview{
		CategoryBreakdown: []models.CategorySpend{},
		MonthlyTrends:     []models.MonthlyCashFlow{},
		TopPayees:         []models.PayeeSpend{},
	}

	// 1. Income vs Expense Totals (Excluding internal transfers to prevent double counting!)
	var totalIncome, totalExpense sql.NullFloat64
	err := d.conn.QueryRow(`
		SELECT 
			SUM(CASE WHEN tx_type = 'CREDIT' THEN amount ELSE 0 END),
			SUM(CASE WHEN tx_type = 'DEBIT' THEN amount ELSE 0 END)
		FROM transactions
		WHERE is_transfer = 0 AND is_excluded = 0
	`).Scan(&totalIncome, &totalExpense)
	if err != nil {
		return nil, err
	}

	if totalIncome.Valid {
		overview.TotalIncome = totalIncome.Float64
	}
	if totalExpense.Valid {
		overview.TotalExpense = totalExpense.Float64
	}
	overview.NetSavings = overview.TotalIncome - overview.TotalExpense
	if overview.TotalIncome > 0 {
		overview.SavingsRate = (overview.NetSavings / overview.TotalIncome) * 100.0
	}

	// 2. Count Accounts & Transactions
	_ = d.conn.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&overview.TotalAccounts)
	_ = d.conn.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&overview.TotalTransactions)

	// 3. Category Breakdown (Expenses only, excluding internal transfers)
	catRows, err := d.conn.Query(`
		SELECT 
			COALESCE(c.id, 'uncat') as cat_id,
			COALESCE(c.name, 'Uncategorized') as cat_name,
			COALESCE(c.color_hex, '#94A3B8') as color_hex,
			COALESCE(c.icon, 'HelpCircle') as icon,
			SUM(t.amount) as total_amount,
			COUNT(t.id) as tx_count
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.tx_type = 'DEBIT' AND t.is_transfer = 0 AND t.is_excluded = 0
		GROUP BY cat_id, cat_name, color_hex, icon
		ORDER BY total_amount DESC
	`)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var cs models.CategorySpend
			if err := catRows.Scan(&cs.CategoryID, &cs.CategoryName, &cs.ColorHex, &cs.Icon, &cs.TotalAmount, &cs.TxCount); err == nil {
				if overview.TotalExpense > 0 {
					cs.Percentage = (cs.TotalAmount / overview.TotalExpense) * 100.0
				}
				overview.CategoryBreakdown = append(overview.CategoryBreakdown, cs)
			}
		}
	}

	// 4. Monthly Trends (Excluding internal transfers)
	trendRows, err := d.conn.Query(`
		SELECT 
			strftime('%Y-%m', tx_date) as month,
			SUM(CASE WHEN tx_type = 'CREDIT' THEN amount ELSE 0 END) as income,
			SUM(CASE WHEN tx_type = 'DEBIT' THEN amount ELSE 0 END) as expense
		FROM transactions
		WHERE is_transfer = 0 AND is_excluded = 0
		GROUP BY month
		ORDER BY month ASC
		LIMIT 12
	`)
	if err == nil {
		defer trendRows.Close()
		for trendRows.Next() {
			var m models.MonthlyCashFlow
			if err := trendRows.Scan(&m.Month, &m.Income, &m.Expense); err == nil {
				m.Net = m.Income - m.Expense
				overview.MonthlyTrends = append(overview.MonthlyTrends, m)
			}
		}
	}

	// 5. Top Payees (Excluding internal transfers)
	payeeRows, err := d.conn.Query(`
		SELECT 
			cleaned_payee,
			payment_mode,
			SUM(amount) as total_spent,
			COUNT(id) as tx_count
		FROM transactions
		WHERE tx_type = 'DEBIT' AND cleaned_payee != '' AND is_transfer = 0 AND is_excluded = 0
		GROUP BY cleaned_payee, payment_mode
		ORDER BY total_spent DESC
		LIMIT 10
	`)
	if err == nil {
		defer payeeRows.Close()
		for payeeRows.Next() {
			var p models.PayeeSpend
			if err := payeeRows.Scan(&p.Payee, &p.PaymentMode, &p.TotalSpent, &p.TxCount); err == nil {
				overview.TopPayees = append(overview.TopPayees, p)
			}
		}
	}
	// 6. Bank Liquidity (Total Liquid Balances in Savings / Current / Wallet)
	_ = d.conn.QueryRow(`
		SELECT COALESCE(SUM(current_balance), 0)
		FROM accounts
		WHERE account_type IN ('SAVINGS', 'CURRENT', 'WALLET')
	`).Scan(&overview.TotalBankLiquidity)

	// 7. Credit Card Outstanding Dues (Latest bill per card)
	_ = d.conn.QueryRow(`
		SELECT COALESCE(SUM(b.total_due_amount), 0)
		FROM credit_card_bills b
		WHERE (b.account_id, b.statement_date) IN (
			SELECT account_id, MAX(statement_date)
			FROM credit_card_bills
			GROUP BY account_id
		)
	`).Scan(&overview.TotalCreditDue)

	// 8. Total Sanctioned Credit Limit across Cards
	_ = d.conn.QueryRow(`
		SELECT COALESCE(SUM(credit_limit), 0)
		FROM accounts
		WHERE account_type = 'CREDIT_CARD' AND credit_limit > 0
	`).Scan(&overview.TotalCreditLimit)

	if overview.TotalCreditLimit > 0 {
		overview.CreditUtilizationRate = (overview.TotalCreditDue / overview.TotalCreditLimit) * 100.0
	}

	// 9. Total Cashback Earned across all bills and transactions
	var billCashback, txCashback float64
	_ = d.conn.QueryRow(`SELECT COALESCE(SUM(cashback_earned), 0) FROM credit_card_bills`).Scan(&billCashback)
	_ = d.conn.QueryRow(`SELECT COALESCE(SUM(cashback_amount), 0) FROM transactions`).Scan(&txCashback)
	overview.TotalCashbackEarned = billCashback + txCashback

	// 10. Total Reward Points Balance across Cards
	_ = d.conn.QueryRow(`
		SELECT COALESCE(SUM(b.reward_points_balance), 0)
		FROM credit_card_bills b
		WHERE (b.account_id, b.statement_date) IN (
			SELECT account_id, MAX(statement_date)
			FROM credit_card_bills
			GROUP BY account_id
		)
	`).Scan(&overview.TotalRewardPoints)

	// 11. Upcoming Active Credit Card Bills
	if bills, err := d.ListCreditCardBills(""); err == nil && len(bills) > 0 {
		overview.UpcomingBills = bills
	}

	return overview, nil
}

func (d *DB) GetPath() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.path
}

func (d *DB) GetDatabaseInfo() (*models.DatabaseInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	absPath, err := filepath.Abs(d.path)
	if err != nil {
		absPath = d.path
	}

	var fileSize int64
	var lastModified time.Time
	if fi, err := os.Stat(d.path); err == nil {
		fileSize = fi.Size()
		lastModified = fi.ModTime()
	}

	info := &models.DatabaseInfo{
		Path:         absPath,
		FileSize:     fileSize,
		WALMode:      true,
		LastModified: lastModified,
	}

	_ = d.conn.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&info.TotalAccounts)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&info.TotalTransactions)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM statement_imports").Scan(&info.TotalStatements)
	_ = d.conn.QueryRow("SELECT COUNT(*) FROM categorization_rules").Scan(&info.TotalRules)

	return info, nil
}

func (d *DB) BackupTo(targetPath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Checkpoint WAL first to flush all dirty pages to the main DB file
	_, _ = d.conn.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	// Attempt modern SQLite VACUUM INTO
	_ = os.Remove(targetPath)
	_, err := d.conn.Exec(fmt.Sprintf("VACUUM INTO '%s'", strings.ReplaceAll(targetPath, "'", "''")))
	if err != nil {
		// Fallback to file copying
		srcFile, err := os.Open(d.path)
		if err != nil {
			return fmt.Errorf("failed to open source db: %w", err)
		}
		defer srcFile.Close()

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create backup file: %w", err)
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy database file: %w", err)
		}
	}

	return nil
}

func (d *DB) RestoreFrom(r io.Reader) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.clearMCPAccess(); err != nil {
		return err
	}

	tempFile := d.path + ".restore.tmp"
	defer os.Remove(tempFile)

	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp restore file: %w", err)
	}

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return fmt.Errorf("failed to write uploaded database: %w", err)
	}
	f.Close()

	// Validate SQLite file header (16 bytes: "SQLite format 3\x00")
	header := make([]byte, 16)
	tempRead, err := os.Open(tempFile)
	if err != nil {
		return fmt.Errorf("failed to read temp db: %w", err)
	}
	n, err := tempRead.Read(header)
	tempRead.Close()
	if err != nil || n < 16 || string(header) != "SQLite format 3\x00" {
		return fmt.Errorf("invalid file format: not a valid SQLite database")
	}

	// Validate by opening test connection and querying tables
	testConn, err := sql.Open("sqlite", tempFile+"?_pragma=busy_timeout(3000)")
	if err != nil {
		return fmt.Errorf("failed to open test connection: %w", err)
	}
	var count int
	err = testConn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	testConn.Close()
	if err != nil || count == 0 {
		return fmt.Errorf("corrupted or empty SQLite database file: %w", err)
	}

	// Safely close active connection
	if d.conn != nil {
		_ = d.conn.Close()
	}

	// Backup current database file
	_ = os.Rename(d.path, d.path+".bak")
	_ = os.Remove(d.path + "-wal")
	_ = os.Remove(d.path + "-shm")

	// Move new file into place
	if err := os.Rename(tempFile, d.path); err != nil {
		// Try to restore original backup if move failed
		_ = os.Rename(d.path+".bak", d.path)
		return fmt.Errorf("failed to replace database file: %w", err)
	}

	// Re-open DB connection
	newConn, err := sql.Open("sqlite", d.path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return fmt.Errorf("failed to re-open database after restore: %w", err)
	}
	newConn.SetMaxOpenConns(1)
	d.conn = newConn

	// Run migrations to ensure restored DB is up to date
	if err := d.migrate(); err != nil {
		return fmt.Errorf("restored database migration failed: %w", err)
	}

	// Restored credentials must never reactivate MCP access.
	if err := d.clearMCPAccess(); err != nil {
		return err
	}

	// Ensure seed categories & rules exist
	_ = d.seedDefaultCategories()
	_ = d.seedDefaultRules()

	return nil
}

func (d *DB) ExportAllDataJSON() (*models.FullExportData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	accounts, err := d.ListAccounts()
	if err != nil {
		return nil, err
	}
	txs, _, err := d.ListTransactions(TransactionFilter{Limit: 100000})
	if err != nil {
		return nil, err
	}
	bills, err := d.ListCreditCardBills("")
	if err != nil {
		return nil, err
	}
	stmts, err := d.ListStatementImports()
	if err != nil {
		return nil, err
	}
	categories, err := d.ListCategories()
	if err != nil {
		return nil, err
	}
	rules, err := d.ListRules()
	if err != nil {
		return nil, err
	}

	return &models.FullExportData{
		ExportedAt:       time.Now(),
		Version:          "1.0",
		Accounts:         accounts,
		Transactions:     txs,
		CreditCardBills:  bills,
		StatementImports: stmts,
		Categories:       categories,
		Rules:            rules,
	}, nil
}

func (d *DB) ExportTransactionsCSV(w io.Writer) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	txs, _, err := d.ListTransactions(TransactionFilter{Limit: 100000})
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"Date", "Account", "Type", "Amount", "Payee", "Mode",
		"Category", "Reference Number", "Raw Narration", "Cashback", "Reward Points",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, t := range txs {
		catName := ""
		if t.CategoryName != nil {
			catName = *t.CategoryName
		}
		row := []string{
			t.TxDate,
			t.AccountName,
			string(t.TxType),
			fmt.Sprintf("%.2f", t.Amount),
			t.CleanedPayee,
			string(t.PaymentMode),
			catName,
			t.ReferenceNumber,
			t.RawNarration,
			fmt.Sprintf("%.2f", t.CashbackAmount),
			fmt.Sprintf("%.2f", t.RewardPointsEarned),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// -------------------------------------------------------------
// Credit Card Rewards, Optimization & Best-Card Engine
// -------------------------------------------------------------

func (d *DB) ListCardRewardRules(accountID string) ([]models.CardRewardRule, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, account_id, merchant_pattern, category_name, reward_percentage,
		       reward_description, max_cap_per_month, min_spend_per_txn, created_at, updated_at
		FROM card_reward_rules
	`
	var args []interface{}
	if accountID != "" {
		query += ` WHERE account_id = ?`
		args = append(args, accountID)
	}
	query += ` ORDER BY reward_percentage DESC`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.CardRewardRule
	for rows.Next() {
		var r models.CardRewardRule
		var maxCap, minSpend sql.NullFloat64
		if err := rows.Scan(&r.ID, &r.AccountID, &r.MerchantPattern, &r.CategoryName,
			&r.RewardPercentage, &r.RewardDescription, &maxCap, &minSpend, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if maxCap.Valid {
			r.MaxCapPerMonth = &maxCap.Float64
		}
		if minSpend.Valid {
			r.MinSpendPerTxn = &minSpend.Float64
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (d *DB) CreateCardRewardRule(r *models.CardRewardRule) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()

	_, err := d.conn.Exec(`
		INSERT INTO card_reward_rules (
			id, account_id, merchant_pattern, category_name, reward_percentage,
			reward_description, max_cap_per_month, min_spend_per_txn, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.AccountID, strings.ToUpper(r.MerchantPattern), r.CategoryName, r.RewardPercentage,
		r.RewardDescription, r.MaxCapPerMonth, r.MinSpendPerTxn, r.CreatedAt, r.UpdatedAt)
	return err
}

func (d *DB) UpdateCardRewardRule(r *models.CardRewardRule) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	r.UpdatedAt = time.Now()
	_, err := d.conn.Exec(`
		UPDATE card_reward_rules SET
			merchant_pattern = ?, category_name = ?, reward_percentage = ?,
			reward_description = ?, max_cap_per_month = ?, min_spend_per_txn = ?, updated_at = ?
		WHERE id = ?
	`, strings.ToUpper(r.MerchantPattern), r.CategoryName, r.RewardPercentage,
		r.RewardDescription, r.MaxCapPerMonth, r.MinSpendPerTxn, r.UpdatedAt, r.ID)
	return err
}

func (d *DB) DeleteCardRewardRule(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec(`DELETE FROM card_reward_rules WHERE id = ?`, id)
	return err
}

func (d *DB) UpdateCardMetadata(accountID string, req models.UpdateCardMetadataRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var updates []string
	var args []interface{}

	if req.CardVariant != nil {
		updates = append(updates, "card_variant = ?")
		args = append(args, *req.CardVariant)
	}
	if req.CardNetwork != nil {
		updates = append(updates, "card_network = ?")
		args = append(args, *req.CardNetwork)
	}
	if req.CardColor != nil {
		updates = append(updates, "card_color = ?")
		args = append(args, *req.CardColor)
	}
	if req.AccountHolderName != nil {
		updates = append(updates, "account_holder_name = ?")
		args = append(args, *req.AccountHolderName)
	}
	if req.CreditLimit != nil {
		updates = append(updates, "credit_limit = ?")
		args = append(args, *req.CreditLimit)
	}
	if req.AnnualFee != nil {
		updates = append(updates, "annual_fee = ?")
		args = append(args, *req.AnnualFee)
	}
	if req.FeeWaiverThreshold != nil {
		updates = append(updates, "fee_waiver_threshold = ?")
		args = append(args, *req.FeeWaiverThreshold)
	}
	if req.BillingDay != nil {
		updates = append(updates, "billing_day = ?")
		args = append(args, *req.BillingDay)
	}
	if req.PaymentDueDays != nil {
		updates = append(updates, "payment_due_days = ?")
		args = append(args, *req.PaymentDueDays)
	}
	if req.BaseRewardRate != nil {
		updates = append(updates, "base_reward_rate = ?")
		args = append(args, *req.BaseRewardRate)
	}
	if req.RewardType != nil {
		updates = append(updates, "reward_type = ?")
		args = append(args, *req.RewardType)
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, accountID)

	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = ?", strings.Join(updates, ", "))
	_, err := d.conn.Exec(query, args...)
	return err
}

func (d *DB) UpdateAccount(accountID string, req models.UpdateAccountRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var updates []string
	var args []interface{}

	if req.BankName != nil && *req.BankName != "" {
		updates = append(updates, "bank_name = ?")
		args = append(args, *req.BankName)
	}
	if req.AccountType != nil && *req.AccountType != "" {
		updates = append(updates, "account_type = ?")
		args = append(args, string(*req.AccountType))
	}
	if req.AccountNumber != nil {
		updates = append(updates, "account_number = ?")
		args = append(args, *req.AccountNumber)
	}
	if req.AccountNumberMask != nil {
		updates = append(updates, "account_number_mask = ?")
		args = append(args, *req.AccountNumberMask)
	}
	if req.Nickname != nil {
		updates = append(updates, "nickname = ?")
		args = append(args, *req.Nickname)
	}
	if req.AccountHolderName != nil {
		updates = append(updates, "account_holder_name = ?")
		args = append(args, *req.AccountHolderName)
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, accountID)

	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = ?", strings.Join(updates, ", "))
	_, err := d.conn.Exec(query, args...)
	return err
}

func (d *DB) SeedDefaultCardRulesIfEmpty(accountID string, variant string, bank string) {
	var count int
	_ = d.conn.QueryRow(`SELECT COUNT(*) FROM card_reward_rules WHERE account_id = ?`, accountID).Scan(&count)
	if count > 0 {
		return
	}

	v := strings.ToUpper(variant + " " + bank)
	var defaultRules []models.CardRewardRule

	if strings.Contains(v, "SWIGGY") {
		_, _ = d.conn.Exec(`UPDATE accounts SET card_color = '#EA580C', reward_type = 'CASHBACK', base_reward_rate = 1.0, annual_fee = 500, fee_waiver_threshold = 200000, billing_day = 12 WHERE id = ?`, accountID)
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "SWIGGY|DINEOUT|INSTAMART|GENIE", CategoryName: "Food & Dining", RewardPercentage: 10.0, RewardDescription: "10% Cashback on Swiggy, Food Delivery, Instamart & Dineout", MaxCapPerMonth: floatPtr(1500)},
			{MerchantPattern: "AMAZON|FLIPKART|MYNTRA|BLINKIT|ZEPTO|UBER|OLA|NYKAA|BIGBASKET|CULT.FIT|BOOKMYSHOW", CategoryName: "Online Shopping", RewardPercentage: 5.0, RewardDescription: "5% Cashback on top E-Commerce & quick commerce platforms", MaxCapPerMonth: floatPtr(1500)},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.0, RewardDescription: "1% Unlimited Cashback on all other retail spends"},
		}
	} else if strings.Contains(v, "AMAZON") || (strings.Contains(v, "ICICI") && strings.Contains(v, "PAY")) {
		_, _ = d.conn.Exec(`UPDATE accounts SET card_color = '#1E293B', reward_type = 'CASHBACK', base_reward_rate = 1.0, annual_fee = 0, fee_waiver_threshold = 0, billing_day = 13 WHERE id = ?`, accountID)
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "AMAZON|AMAZON.IN|AMAZON PAY", CategoryName: "Shopping", RewardPercentage: 5.0, RewardDescription: "5% Unlimited Cashback on Amazon Shopping for Prime members"},
			{MerchantPattern: "RECHARGE|BILL|ELECTRICITY|GAS|DTH|POSTPAID|BROADBAND|INSURANCE|SWIGGY|ZOMATO|UBER", CategoryName: "Utilities & Payments", RewardPercentage: 2.0, RewardDescription: "2% Unlimited Cashback on Amazon Pay utilities & 100+ partner merchants"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.0, RewardDescription: "1% Unlimited Cashback on all other retail & dining spends"},
		}
	} else if strings.Contains(v, "FLIPKART") || strings.Contains(v, "AXIS") {
		_, _ = d.conn.Exec(`UPDATE accounts SET card_color = '#0284C7', reward_type = 'CASHBACK', base_reward_rate = 1.5, annual_fee = 500, fee_waiver_threshold = 350000, billing_day = 15 WHERE id = ?`, accountID)
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "FLIPKART|MYNTRA|SHOPSY", CategoryName: "Shopping", RewardPercentage: 5.0, RewardDescription: "5% Unlimited Cashback on Flipkart, Myntra & Cleartrip"},
			{MerchantPattern: "SWIGGY|UBER|PVR|CULT.FIT|TATA 1MG", CategoryName: "Preferred Partners", RewardPercentage: 4.0, RewardDescription: "4% Unlimited Cashback on preferred merchant partners"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.5, RewardDescription: "1.5% Unlimited Cashback on all other eligible spends"},
		}
	} else if strings.Contains(v, "REGALIA") || strings.Contains(v, "DINERS") {
		_, _ = d.conn.Exec(`UPDATE accounts SET card_color = '#1E3A8A', reward_type = 'REWARD_POINTS', base_reward_rate = 2.67, annual_fee = 2500, fee_waiver_threshold = 300000, billing_day = 16 WHERE id = ?`, accountID)
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "SMARTBUY|FLIGHT|HOTEL|CLEARTRIP|YATRA", CategoryName: "Travel & Flights", RewardPercentage: 13.3, RewardDescription: "5X Reward Points (13.3% value) on SmartBuy Flights & Hotels"},
			{MerchantPattern: "DINING|RESTAURANT|ZOMATO|SWIGGY", CategoryName: "Dining", RewardPercentage: 4.0, RewardDescription: "2X Reward Points on Dining spends"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "General Spends", RewardPercentage: 2.67, RewardDescription: "4 Reward Points per ₹150 spent across all categories"},
		}
	} else if strings.Contains(v, "RUPAY") {
		_, _ = d.conn.Exec(`UPDATE accounts SET card_color = '#831843', reward_type = 'CASHBACK', base_reward_rate = 1.0, annual_fee = 250, fee_waiver_threshold = 50000, billing_day = 1 WHERE id = ?`, accountID)
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "UPI|PAYTM|PHONEPE|GPAY|BHIM", CategoryName: "UPI Payments", RewardPercentage: 3.0, RewardDescription: "3% CashPoints on UPI Merchant payments"},
			{MerchantPattern: "SUPERMARKET|GROCERY|DINING", CategoryName: "Groceries & Dining", RewardPercentage: 2.0, RewardDescription: "2% CashPoints on Groceries & Dining"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.0, RewardDescription: "1% CashPoints on all other spends"},
		}
	} else {
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "ALL_OTHER", CategoryName: "General Spends", RewardPercentage: 1.0, RewardDescription: "1% Base reward on eligible spends"},
		}
	}

	for _, r := range defaultRules {
		r.ID = uuid.New().String()
		r.AccountID = accountID
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
		_, _ = d.conn.Exec(`
			INSERT INTO card_reward_rules (
				id, account_id, merchant_pattern, category_name, reward_percentage,
				reward_description, max_cap_per_month, min_spend_per_txn, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, r.ID, r.AccountID, r.MerchantPattern, r.CategoryName, r.RewardPercentage, r.RewardDescription, r.MaxCapPerMonth, r.MinSpendPerTxn, r.CreatedAt, r.UpdatedAt)
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

func (d *DB) GetCardPortfolioOverview() (*models.CardPortfolioOverview, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT id, bank_name, account_type, account_number_mask, currency, opening_balance, current_balance,
		       credit_limit, billing_cycle_day, nickname, customer_id, ifsc_code, branch_name,
		       card_network, card_variant, annual_fee, fee_waiver_threshold, billing_day, payment_due_days,
		       card_color, reward_type, base_reward_rate, account_holder_name, created_at, updated_at
		FROM accounts
		WHERE account_type = 'CREDIT_CARD'
		ORDER BY bank_name ASC
	`)
	if err != nil {
		return nil, err
	}

	type rawCardRow struct {
		a           models.Account
		annFee      sql.NullFloat64
		feeWaiver   sql.NullFloat64
		billDay     sql.NullInt32
		payDueDays  sql.NullInt32
		cardColor   sql.NullString
		rewType     sql.NullString
		baseRewRate sql.NullFloat64
	}

	var rawRows []rawCardRow
	for rows.Next() {
		var a models.Account
		var credLimit sql.NullFloat64
		var billCycle sql.NullInt32
		var nickname, custID, ifsc, branch, cardNet, cardVar, cardColor, rewType, holderName sql.NullString
		var annFee, feeWaiver, baseRewRate sql.NullFloat64
		var billDay, payDueDays sql.NullInt32

		if err := rows.Scan(
			&a.ID, &a.BankName, &a.AccountType, &a.AccountNumberMask, &a.Currency, &a.OpeningBalance, &a.CurrentBalance,
			&credLimit, &billCycle, &nickname, &custID, &ifsc, &branch,
			&cardNet, &cardVar, &annFee, &feeWaiver, &billDay, &payDueDays,
			&cardColor, &rewType, &baseRewRate, &holderName, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}

		if credLimit.Valid { a.CreditLimit = &credLimit.Float64 }
		if billCycle.Valid { val := int(billCycle.Int32); a.BillingCycleDay = &val }
		if nickname.Valid { a.Nickname = &nickname.String }
		if custID.Valid { a.CustomerID = &custID.String }
		if ifsc.Valid { a.IFSCCode = &ifsc.String }
		if branch.Valid { a.BranchName = &branch.String }
		if cardNet.Valid { a.CardNetwork = &cardNet.String }
		if cardVar.Valid { a.CardVariant = &cardVar.String }
		if holderName.Valid { a.AccountHolderName = &holderName.String }

		rawRows = append(rawRows, rawCardRow{
			a:           a,
			annFee:      annFee,
			feeWaiver:   feeWaiver,
			billDay:     billDay,
			payDueDays:  payDueDays,
			cardColor:   cardColor,
			rewType:     rewType,
			baseRewRate: baseRewRate,
		})
	}
	rows.Close()

	now := time.Now()
	currentYear := now.Year()

	cards := make([]models.CardDetails, 0)
	var totalCreditLimit, totalOutstanding, totalAnnualFees, totalFeeSavings, totalCashback, totalRewardPoints float64

	for _, r := range rawRows {
		var cd models.CardDetails
		cd.Account = r.a
		cd.RewardRules = make([]models.CardRewardRule, 0)
		if r.annFee.Valid { cd.AnnualFee = r.annFee.Float64 }
		if r.feeWaiver.Valid { cd.FeeWaiverThreshold = r.feeWaiver.Float64 }
		if r.billDay.Valid && r.billDay.Int32 > 0 { cd.BillingDay = int(r.billDay.Int32) } else { cd.BillingDay = 12 }
		if r.payDueDays.Valid && r.payDueDays.Int32 > 0 { cd.PaymentDueDays = int(r.payDueDays.Int32) } else { cd.PaymentDueDays = 20 }
		if r.cardColor.Valid && r.cardColor.String != "" { cd.CardColor = r.cardColor.String } else { cd.CardColor = "#1E293B" }
		if r.rewType.Valid && r.rewType.String != "" { cd.RewardType = r.rewType.String } else { cd.RewardType = "CASHBACK" }
		if r.baseRewRate.Valid && r.baseRewRate.Float64 > 0 { cd.BaseRewardRate = r.baseRewRate.Float64 } else { cd.BaseRewardRate = 1.0 }

		// Fetch YTD spend
		var ytdSpend float64
		_ = d.conn.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE account_id = ? AND tx_type = 'DEBIT' AND strftime('%Y', tx_date) = ? AND is_transfer = 0 AND is_excluded = 0
		`, r.a.ID, fmt.Sprintf("%d", currentYear)).Scan(&ytdSpend)
		cd.TotalSpendThisYear = ytdSpend

		// Fee waiver progress
		if cd.FeeWaiverThreshold > 0 {
			cd.FeeWaiverProgressPct = (cd.TotalSpendThisYear / cd.FeeWaiverThreshold) * 100
			if cd.FeeWaiverProgressPct > 100 {
				cd.FeeWaiverProgressPct = 100
			}
			cd.FeeWaiverRemaining = cd.FeeWaiverThreshold - cd.TotalSpendThisYear
			if cd.FeeWaiverRemaining < 0 {
				cd.FeeWaiverRemaining = 0
			}
			if cd.TotalSpendThisYear >= cd.FeeWaiverThreshold {
				totalFeeSavings += cd.AnnualFee
			}
		}
		totalAnnualFees += cd.AnnualFee

		// Fetch latest credit card bill metadata
		var totalDue, availLimit, cbEarned, rewPts float64
		_ = d.conn.QueryRow(`
			SELECT COALESCE(total_due_amount, 0), COALESCE(available_credit_limit, 0), COALESCE(cashback_earned, 0), COALESCE(reward_points_balance, 0)
			FROM credit_card_bills
			WHERE account_id = ?
			ORDER BY statement_date DESC
			LIMIT 1
		`, r.a.ID).Scan(&totalDue, &availLimit, &cbEarned, &rewPts)
		cd.TotalDueAmount = totalDue
		cd.AvailableCreditLimit = availLimit
		cd.TotalCashbackEarned = cbEarned
		cd.TotalRewardPoints = rewPts

		totalCashback += cbEarned
		totalRewardPoints += rewPts
		if r.a.CreditLimit != nil {
			totalCreditLimit += *r.a.CreditLimit
		}
		totalOutstanding += r.a.CurrentBalance

		// Calculate interest-free runway
		bDay := cd.BillingDay
		if bDay < 1 || bDay > 31 {
			bDay = 12
		}

		// Next statement date
		currentDay := now.Day()
		var nextStmtDate time.Time
		if currentDay <= bDay {
			nextStmtDate = time.Date(now.Year(), now.Month(), bDay, 0, 0, 0, 0, now.Location())
		} else {
			nextMonth := now.AddDate(0, 1, 0)
			nextStmtDate = time.Date(nextMonth.Year(), nextMonth.Month(), bDay, 0, 0, 0, 0, now.Location())
		}
		daysUntilStmt := int(nextStmtDate.Sub(now).Hours() / 24)
		if daysUntilStmt < 0 {
			daysUntilStmt = 0
		}
		cd.DaysUntilStatement = daysUntilStmt

		// Payment due date is statement date + PaymentDueDays
		nextPaymentDueDate := nextStmtDate.AddDate(0, 0, cd.PaymentDueDays)
		interestFreeDays := int(nextPaymentDueDate.Sub(now).Hours() / 24)
		if interestFreeDays < 15 {
			interestFreeDays = 15
		}
		cd.InterestFreeDaysRemaining = interestFreeDays
		cd.NextStatementDate = nextStmtDate.Format("2006-01-02")
		cd.NextPaymentDueDate = nextPaymentDueDate.Format("2006-01-02")

		// Fetch rules
		rulesRows, _ := d.conn.Query(`
			SELECT id, account_id, merchant_pattern, category_name, reward_percentage,
			       reward_description, max_cap_per_month, min_spend_per_txn, created_at, updated_at
			FROM card_reward_rules
			WHERE account_id = ?
			ORDER BY reward_percentage DESC
		`, r.a.ID)
		if rulesRows != nil {
			for rulesRows.Next() {
				var rule models.CardRewardRule
				var maxC, minS sql.NullFloat64
				if err := rulesRows.Scan(&rule.ID, &rule.AccountID, &rule.MerchantPattern, &rule.CategoryName,
					&rule.RewardPercentage, &rule.RewardDescription, &maxC, &minS, &rule.CreatedAt, &rule.UpdatedAt); err == nil {
					if maxC.Valid { rule.MaxCapPerMonth = &maxC.Float64 }
					if minS.Valid { rule.MinSpendPerTxn = &minS.Float64 }
					cd.RewardRules = append(cd.RewardRules, rule)
				}
			}
			rulesRows.Close()
		}

		cards = append(cards, cd)
	}

	// Determine best card to swipe today (highest interest-free days runway)
	var bestCard *models.CardDetails
	maxRunway := -1
	for i := range cards {
		if cards[i].InterestFreeDaysRemaining > maxRunway {
			maxRunway = cards[i].InterestFreeDaysRemaining
			bestCard = &cards[i]
		}
	}

	var overallUtil float64
	if totalCreditLimit > 0 {
		overallUtil = (totalOutstanding / totalCreditLimit) * 100
	}

	return &models.CardPortfolioOverview{
		Cards:                    cards,
		BestCardToSwipeToday:     bestCard,
		TotalCreditLimit:         totalCreditLimit,
		TotalOutstanding:         totalOutstanding,
		OverallUtilizationRate:   overallUtil,
		TotalAnnualFeeLiability:  totalAnnualFees,
		TotalFeeSavingsProjected: totalFeeSavings,
		TotalCashbackEarned:      totalCashback,
		TotalRewardPoints:        totalRewardPoints,
	}, nil
}

func (d *DB) GetCategoryBudgetSummary(monthStr string) (*models.BudgetSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()
	currentMonthStr := now.Format("2006-01")

	if monthStr == "" {
		monthStr = currentMonthStr
	}

	parsedTime, err := time.Parse("2006-01", monthStr)
	if err != nil {
		parsedTime = now
		monthStr = currentMonthStr
	}

	year := parsedTime.Year()
	month := parsedTime.Month()

	// Total days in month
	totalDays := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	var daysElapsed, daysRemaining int
	if monthStr == currentMonthStr {
		daysElapsed = now.Day()
		daysRemaining = totalDays - daysElapsed
		if daysRemaining < 0 {
			daysRemaining = 0
		}
	} else if monthStr < currentMonthStr {
		daysElapsed = totalDays
		daysRemaining = 0
	} else {
		daysElapsed = 0
		daysRemaining = totalDays
	}

	type catRow struct {
		ID            string
		Name          string
		Color         sql.NullString
		Icon          sql.NullString
		MonthlyBudget sql.NullFloat64
		BudgetID      sql.NullString
		CustomLimit   sql.NullFloat64
	}

	rows, err := d.conn.Query(`
		SELECT c.id, c.name, c.color_hex, c.icon, c.monthly_budget, cb.id, cb.monthly_limit
		FROM categories c
		LEFT JOIN category_budgets cb ON c.id = cb.category_id AND cb.month = ?
		ORDER BY c.name ASC
	`, monthStr)
	if err != nil {
		return nil, err
	}

	var catList []catRow
	for rows.Next() {
		var cr catRow
		if err := rows.Scan(&cr.ID, &cr.Name, &cr.Color, &cr.Icon, &cr.MonthlyBudget, &cr.BudgetID, &cr.CustomLimit); err == nil {
			catList = append(catList, cr)
		}
	}
	rows.Close()

	budgets := make([]models.CategoryBudget, 0)
	var totalBudget, totalSpent, totalRemaining float64
	var overspentCount, warningCount int

	for _, c := range catList {
		var limit float64
		if c.CustomLimit.Valid && c.CustomLimit.Float64 > 0 {
			limit = c.CustomLimit.Float64
		} else if c.MonthlyBudget.Valid && c.MonthlyBudget.Float64 > 0 {
			limit = c.MonthlyBudget.Float64
		}

		// Fetch actual spend for this category in this month
		var spend float64
		_ = d.conn.QueryRow(`
			SELECT COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE category_id = ? AND tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 AND strftime('%Y-%m', tx_date) = ?
		`, c.ID, monthStr).Scan(&spend)

		// Calculate metrics
		var spentPct, remAmt, dailyAllowance, projectedSpend float64
		status := "SAFE"
		pacingStatus := "ON_TRACK"

		if limit > 0 {
			spentPct = (spend / limit) * 100
			remAmt = limit - spend

			if spentPct > 100 {
				status = "EXCEEDED"
				overspentCount++
			} else if spentPct >= 80 {
				status = "WARNING"
				warningCount++
			}

			if daysRemaining > 0 && remAmt > 0 {
				dailyAllowance = remAmt / float64(daysRemaining)
			}

			// Pacing estimation
			if daysElapsed > 0 {
				dailyAvg := spend / float64(daysElapsed)
				projectedSpend = dailyAvg * float64(totalDays)
			} else {
				projectedSpend = 0
			}

			if projectedSpend > limit {
				if spend > limit {
					pacingStatus = "PACING_EXCEEDED"
				} else {
					pacingStatus = "PACING_HIGH"
				}
			}

			totalBudget += limit
			totalSpent += spend
			if remAmt > 0 {
				totalRemaining += remAmt
			}
		} else {
			// Even if limit is 0, we track spend if any
			if spend > 0 {
				totalSpent += spend
			}
		}

		var colorStr, iconStr string
		if c.Color.Valid && c.Color.String != "" {
			colorStr = c.Color.String
		} else {
			colorStr = "#64748B"
		}
		if c.Icon.Valid {
			iconStr = c.Icon.String
		}

		b := models.CategoryBudget{
			ID:                        c.ID,
			CategoryID:                c.ID,
			CategoryName:              c.Name,
			CategoryColor:             colorStr,
			CategoryIcon:              iconStr,
			MonthlyLimit:              limit,
			Month:                     monthStr,
			ActualSpend:               spend,
			SpentPercentage:           spentPct,
			RemainingAmount:           remAmt,
			Status:                    status,
			DaysElapsedInMonth:        daysElapsed,
			DaysRemainingInMonth:      daysRemaining,
			TotalDaysInMonth:          totalDays,
			DailyRecommendedAllowance: dailyAllowance,
			ProjectedMonthEndSpend:    projectedSpend,
			PacingStatus:              pacingStatus,
		}
		budgets = append(budgets, b)
	}

	var overallSpentPct, overallDailyAllowance float64
	if totalBudget > 0 {
		overallSpentPct = (totalSpent / totalBudget) * 100
		if totalRemaining > 0 && daysRemaining > 0 {
			overallDailyAllowance = totalRemaining / float64(daysRemaining)
		}
	}

	categoriesWithBudgets := 0
	for _, b := range budgets {
		if b.MonthlyLimit > 0 {
			categoriesWithBudgets++
		}
	}

	return &models.BudgetSummary{
		Month:                            monthStr,
		TotalBudget:                      totalBudget,
		TotalSpent:                       totalSpent,
		TotalRemaining:                   totalRemaining,
		OverallSpentPercentage:           overallSpentPct,
		TotalDaysInMonth:                 totalDays,
		DaysElapsedInMonth:               daysElapsed,
		DaysRemainingInMonth:             daysRemaining,
		OverallDailyRecommendedAllowance: overallDailyAllowance,
		CategoriesWithBudgets:            categoriesWithBudgets,
		OverspentCategoriesCount:         overspentCount,
		WarningCategoriesCount:           warningCount,
		Budgets:                          budgets,
	}, nil
}

func (d *DB) UpsertCategoryBudget(categoryID string, monthStr string, limit float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if monthStr == "" {
		monthStr = time.Now().Format("2006-01")
	}

	id := uuid.New().String()
	_, err := d.conn.Exec(`
		INSERT INTO category_budgets (id, category_id, month, monthly_limit, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(category_id, month) DO UPDATE SET monthly_limit = excluded.monthly_limit, updated_at = CURRENT_TIMESTAMP
	`, id, categoryID, monthStr, limit)
	if err != nil {
		return err
	}

	// Also update base category default budget
	_, err = d.conn.Exec(`UPDATE categories SET monthly_budget = ? WHERE id = ?`, limit, categoryID)
	return err
}

func (d *DB) DeleteCategoryBudget(categoryID string, monthStr string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if monthStr != "" {
		_, err := d.conn.Exec(`DELETE FROM category_budgets WHERE category_id = ? AND month = ?`, categoryID, monthStr)
		if err != nil {
			return err
		}
	} else {
		_, err := d.conn.Exec(`DELETE FROM category_budgets WHERE category_id = ?`, categoryID)
		if err != nil {
			return err
		}
	}

	_, err := d.conn.Exec(`UPDATE categories SET monthly_budget = 0 WHERE id = ?`, categoryID)
	return err
}

func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.conn.Query(query, args...)
}

func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.conn.Exec(query, args...)
}

// ==========================================
// TRANSFER RECONCILIATION METHODS
// ==========================================

func (d *DB) LinkTransferPair(debitTxID, creditTxID, reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if reason == "" {
		reason = "CC_BILL_PAYMENT_MATCH"
	}

	_, err = tx.Exec(`
		UPDATE transactions 
		SET is_transfer = 1, transfer_peer_id = ?, transfer_match_reason = ? 
		WHERE id = ?
	`, creditTxID, reason, debitTxID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE transactions 
		SET is_transfer = 1, transfer_peer_id = ?, transfer_match_reason = ? 
		WHERE id = ?
	`, debitTxID, reason, creditTxID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) UnlinkTransferPair(txID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var peerID sql.NullString
	_ = d.conn.QueryRow(`SELECT transfer_peer_id FROM transactions WHERE id = ?`, txID).Scan(&peerID)

	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE transactions 
		SET is_transfer = 0, transfer_peer_id = NULL, transfer_match_reason = NULL 
		WHERE id = ?
	`, txID)
	if err != nil {
		return err
	}

	if peerID.Valid && peerID.String != "" {
		_, err = tx.Exec(`
			UPDATE transactions 
			SET is_transfer = 0, transfer_peer_id = NULL, transfer_match_reason = NULL 
			WHERE id = ?
		`, peerID.String)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) ToggleExcludeTransaction(txID string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var currentExcluded bool
	err := d.conn.QueryRow(`SELECT is_excluded FROM transactions WHERE id = ?`, txID).Scan(&currentExcluded)
	if err != nil {
		return false, err
	}

	newStatus := !currentExcluded
	_, err = d.conn.Exec(`UPDATE transactions SET is_excluded = ? WHERE id = ?`, newStatus, txID)
	return newStatus, err
}

// ==========================================
// MERCHANT & PAYEE INTELLIGENCE METHODS
// ==========================================

func (d *DB) ListMerchants(search, category, sortBy string) (*models.MerchantListResponse, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	where := []string{"t.cleaned_payee != ''", "t.is_transfer = 0", "t.is_excluded = 0"}
	args := []interface{}{}

	if search != "" {
		where = append(where, "(t.cleaned_payee LIKE ? OR t.raw_narration LIKE ?)")
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
	}
	if category != "" {
		where = append(where, "c.name = ?")
		args = append(args, category)
	}

	whereClause := strings.Join(where, " AND ")

	orderClause := "total_spend DESC"
	if sortBy == "tx_count" {
		orderClause = "tx_count DESC"
	} else if sortBy == "aov" {
		orderClause = "average_order_value DESC"
	} else if sortBy == "recent" {
		orderClause = "last_tx_date DESC"
	} else if sortBy == "name" {
		orderClause = "t.cleaned_payee ASC"
	}

	query := fmt.Sprintf(`
		SELECT 
			t.cleaned_payee,
			COALESCE(c.name, 'Uncategorized') as category_name,
			COALESCE(c.color_hex, '#64748B') as category_color,
			COALESCE(c.icon, '') as category_icon,
			COALESCE(SUM(CASE WHEN t.tx_type = 'DEBIT' THEN t.amount ELSE 0 END), 0) as total_spend,
			COALESCE(SUM(CASE WHEN t.tx_type = 'CREDIT' THEN t.amount ELSE 0 END), 0) as total_credits,
			COUNT(t.id) as tx_count,
			COALESCE(AVG(CASE WHEN t.tx_type = 'DEBIT' THEN t.amount ELSE NULL END), 0) as average_order_value,
			MIN(t.tx_date) as first_tx_date,
			MAX(t.tx_date) as last_tx_date,
			COALESCE(
				(SELECT a.bank_name FROM transactions st JOIN accounts a ON st.account_id = a.id WHERE st.cleaned_payee = t.cleaned_payee ORDER BY st.tx_date DESC LIMIT 1),
				''
			) as primary_source
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE %s
		GROUP BY t.cleaned_payee
		ORDER BY %s
	`, whereClause, orderClause)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants = make([]models.MerchantSummaryItem, 0)
	var grandTotalSpend float64
	categorySpendMap := make(map[string]float64)

	for rows.Next() {
		var m models.MerchantSummaryItem
		if err := rows.Scan(
			&m.CleanedPayee, &m.CategoryName, &m.CategoryColor, &m.CategoryIcon,
			&m.TotalSpend, &m.TotalCredits, &m.TxCount, &m.AverageOrderValue,
			&m.FirstTxDate, &m.LastTxDate, &m.PrimarySource,
		); err != nil {
			return nil, err
		}

		grandTotalSpend += m.TotalSpend
		categorySpendMap[m.CategoryName] += m.TotalSpend
		merchants = append(merchants, m)
	}

	// Compute spend share %
	topCat := "None"
	topCatSpend := 0.0
	for cat, sp := range categorySpendMap {
		if sp > topCatSpend {
			topCatSpend = sp
			topCat = cat
		}
	}

	for i := range merchants {
		if grandTotalSpend > 0 {
			merchants[i].SpendSharePct = (merchants[i].TotalSpend / grandTotalSpend) * 100
		}
	}

	var overallAOV float64
	if len(merchants) > 0 && grandTotalSpend > 0 {
		totalTx := 0
		for _, m := range merchants {
			totalTx += m.TxCount
		}
		if totalTx > 0 {
			overallAOV = grandTotalSpend / float64(totalTx)
		}
	}

	return &models.MerchantListResponse{
		TotalMerchants:    len(merchants),
		TotalSpend:        grandTotalSpend,
		TopCategory:       topCat,
		AverageOrderValue: overallAOV,
		Merchants:         merchants,
	}, nil
}

func (d *DB) GetMerchantProfile(payeeName string) (*models.MerchantProfile, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 1. Basic Stats
	var p models.MerchantProfile
	p.CleanedPayee = payeeName
	p.PaymentSources = make([]models.PaymentSourceShare, 0)
	p.MonthlySpendHistory = make([]models.MonthlySpendDataPoint, 0)
	p.RecentTransactions = make([]models.Transaction, 0)

	var catName, catColor, catIcon sql.NullString
	var firstDate, lastDate sql.NullString
	var totalSpend, totalCredits, avgOrderVal sql.NullFloat64
	var totalTx, debitTx, creditTx sql.NullInt32

	err := d.conn.QueryRow(`
		SELECT 
			COALESCE(c.name, 'Uncategorized'),
			COALESCE(c.color_hex, '#64748B'),
			COALESCE(c.icon, ''),
			COALESCE(SUM(CASE WHEN t.tx_type = 'DEBIT' THEN t.amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN t.tx_type = 'CREDIT' THEN t.amount ELSE 0 END), 0),
			COUNT(t.id),
			COALESCE(SUM(CASE WHEN t.tx_type = 'DEBIT' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN t.tx_type = 'CREDIT' THEN 1 ELSE 0 END), 0),
			COALESCE(AVG(CASE WHEN t.tx_type = 'DEBIT' THEN t.amount ELSE NULL END), 0),
			MIN(t.tx_date),
			MAX(t.tx_date)
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.cleaned_payee = ?
	`, payeeName).Scan(
		&catName, &catColor, &catIcon,
		&totalSpend, &totalCredits,
		&totalTx, &debitTx, &creditTx,
		&avgOrderVal, &firstDate, &lastDate,
	)
	if err != nil {
		return nil, err
	}

	if catName.Valid { p.CategoryName = catName.String }
	if catColor.Valid { p.CategoryColor = catColor.String }
	if catIcon.Valid { p.CategoryIcon = catIcon.String }
	if totalSpend.Valid { p.TotalSpend = totalSpend.Float64 }
	if totalCredits.Valid { p.TotalCredits = totalCredits.Float64 }
	p.NetSpend = p.TotalSpend - p.TotalCredits
	if totalTx.Valid { p.TotalTxCount = int(totalTx.Int32) }
	if debitTx.Valid { p.DebitTxCount = int(debitTx.Int32) }
	if creditTx.Valid { p.CreditTxCount = int(creditTx.Int32) }
	if avgOrderVal.Valid { p.AverageOrderValue = avgOrderVal.Float64 }
	if firstDate.Valid { p.FirstTxDate = firstDate.String }
	if lastDate.Valid {
		p.LastTxDate = lastDate.String
		if lDate, err := time.Parse("2006-01-02", p.LastTxDate); err == nil {
			p.DaysSinceLastTx = int(math.Max(0, time.Since(lDate).Hours()/24))
		}
	}

	// 2. Preferred Payment Sources Breakdown
	sourceRows, _ := d.conn.Query(`
		SELECT 
			a.bank_name || ' (' || a.account_type || ')' as acc_name,
			SUM(t.amount) as spend,
			COUNT(t.id) as cnt
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.cleaned_payee = ? AND t.tx_type = 'DEBIT'
		GROUP BY acc_name
		ORDER BY spend DESC
	`, payeeName)
	if sourceRows != nil {
		for sourceRows.Next() {
			var s models.PaymentSourceShare
			if err := sourceRows.Scan(&s.AccountName, &s.SpendAmount, &s.TxCount); err == nil {
				if p.TotalSpend > 0 {
					s.SharePct = (s.SpendAmount / p.TotalSpend) * 100
				}
				p.PaymentSources = append(p.PaymentSources, s)
			}
		}
		sourceRows.Close()
	}
	if len(p.PaymentSources) > 0 {
		p.PreferredPaymentSource = p.PaymentSources[0].AccountName
	}

	// 3. Preferred Payment Mode
	var prefMode sql.NullString
	_ = d.conn.QueryRow(`
		SELECT payment_mode FROM transactions WHERE cleaned_payee = ? GROUP BY payment_mode ORDER BY COUNT(*) DESC LIMIT 1
	`, payeeName).Scan(&prefMode)
	if prefMode.Valid {
		p.PreferredPaymentMode = prefMode.String
	}

	// 4. Monthly Spend History
	monthRows, _ := d.conn.Query(`
		SELECT 
			strftime('%Y-%m', tx_date) as m,
			COALESCE(SUM(amount), 0),
			COUNT(id)
		FROM transactions
		WHERE cleaned_payee = ? AND tx_type = 'DEBIT'
		GROUP BY m
		ORDER BY m ASC
	`, payeeName)
	if monthRows != nil {
		for monthRows.Next() {
			var dp models.MonthlySpendDataPoint
			if err := monthRows.Scan(&dp.Month, &dp.SpendAmount, &dp.TxCount); err == nil {
				p.MonthlySpendHistory = append(p.MonthlySpendHistory, dp)
			}
		}
		monthRows.Close()
	}

	// 5. Recent Transactions
	txRows, _ := d.conn.Query(`
		SELECT 
			t.id, t.account_id, a.bank_name || ' (' || a.account_type || ')',
			t.tx_date, t.raw_narration, t.cleaned_payee, t.payment_mode, t.reference_number,
			t.tx_type, t.amount, t.is_transfer, t.is_excluded, t.created_at
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.cleaned_payee = ?
		ORDER BY t.tx_date DESC
		LIMIT 50
	`, payeeName)
	if txRows != nil {
		for txRows.Next() {
			var t models.Transaction
			if err := txRows.Scan(
				&t.ID, &t.AccountID, &t.AccountName,
				&t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.PaymentMode, &t.ReferenceNumber,
				&t.TxType, &t.Amount, &t.IsTransfer, &t.IsExcluded, &t.CreatedAt,
			); err == nil {
				p.RecentTransactions = append(p.RecentTransactions, t)
			}
		}
		txRows.Close()
	}

	return &p, nil
}

// ==========================================
// CASH FLOW SANKEY & MOM INTELLIGENCE ENGINE
// ==========================================

func (d *DB) GetCashFlowIntelligence(period string) (*models.CashFlowIntelligenceResponse, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	res := &models.CashFlowIntelligenceResponse{
		AvailableMonths:     []string{},
		Anomalies:           []models.MoMAnomaly{},
		CategoryComparisons: []models.CategoryComparisonItem{},
		Sankey: models.SankeyData{
			Nodes: []models.SankeyNode{},
			Links: []models.SankeyLink{},
		},
	}

	// 1. Fetch available months from database
	mRows, err := d.conn.Query(`
		SELECT DISTINCT strftime('%Y-%m', tx_date) as m
		FROM transactions
		WHERE is_transfer = 0 AND is_excluded = 0 AND tx_date IS NOT NULL AND tx_date != ''
		ORDER BY m DESC
	`)
	if err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var m string
			if err := mRows.Scan(&m); err == nil && m != "" {
				res.AvailableMonths = append(res.AvailableMonths, m)
			}
		}
	}

	// 2. Resolve Selected Period and Previous Period
	selectedMonth := period
	if selectedMonth == "" || selectedMonth == "current" {
		if len(res.AvailableMonths) > 0 {
			selectedMonth = res.AvailableMonths[0]
		} else {
			selectedMonth = time.Now().Format("2006-01")
		}
	}
	res.Period = selectedMonth
	res.SelectedMonth = selectedMonth

	// Determine Previous Month
	if strings.HasPrefix(selectedMonth, "20") && len(selectedMonth) == 7 {
		t, err := time.Parse("2006-01", selectedMonth)
		if err == nil {
			res.PreviousMonth = t.AddDate(0, -1, 0).Format("2006-01")
		}
	} else if len(res.AvailableMonths) > 1 {
		res.PreviousMonth = res.AvailableMonths[1]
	}

	// 3. Construct SQL Date Clauses
	var dateClause string
	var dateArgs []interface{}

	if selectedMonth == "ALL" {
		dateClause = "1=1"
	} else if selectedMonth == "LAST_3_MONTHS" {
		if len(res.AvailableMonths) >= 3 {
			startM := res.AvailableMonths[2]
			endM := res.AvailableMonths[0]
			dateClause = "tx_date >= ? AND tx_date <= ?"
			dateArgs = append(dateArgs, startM+"-01", endM+"-31")
		} else {
			dateClause = "1=1"
		}
	} else if strings.HasPrefix(selectedMonth, "FY-") {
		// e.g. FY-2025-26
		parts := strings.Split(strings.TrimPrefix(selectedMonth, "FY-"), "-")
		if len(parts) >= 1 {
			startYear := parts[0]
			sy, _ := strconv.Atoi(startYear)
			ey := sy + 1
			dateClause = "tx_date >= ? AND tx_date <= ?"
			dateArgs = append(dateArgs, fmt.Sprintf("%d-04-01", sy), fmt.Sprintf("%d-03-31", ey))
		} else {
			dateClause = "1=1"
		}
	} else {
		// Specific YYYY-MM
		dateClause = "tx_date >= ? AND tx_date <= ?"
		dateArgs = append(dateArgs, selectedMonth+"-01", selectedMonth+"-31")
	}

	// 4. Extract Inflow Credits and build Inflow Nodes & Links (Layer 0 -> Layer 1)
	inflowQuery := fmt.Sprintf(`
		SELECT 
			t.id, t.raw_narration, t.cleaned_payee, t.payment_mode, t.amount,
			a.id as acc_id, a.bank_name, a.account_type, COALESCE(a.account_number_mask, '') as mask,
			COALESCE(a.nickname, '') as nickname, COALESCE(a.card_variant, '') as variant
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.tx_type = 'CREDIT' AND t.is_transfer = 0 AND t.is_excluded = 0 AND %s
	`, dateClause)

	creditRows, err := d.conn.Query(inflowQuery, dateArgs...)
	sourceTotals := make(map[string]float64)       // source_id -> total
	sourceCounts := make(map[string]int)           // source_id -> count
	sourceToAcc := make(map[string]map[string]float64) // source_id -> acc_id -> amount
	accNames := make(map[string]string)            // acc_id -> display_name
	accTypes := make(map[string]string)            // acc_id -> account_type
	accInflow := make(map[string]float64)          // acc_id -> total inflow

	var totalInflow float64

	if err == nil {
		for creditRows.Next() {
			var (
				id, rawNarration, cleanedPayee, payMode string
				amount                                  float64
				accID, bankName, accType, mask, nick, variant string
			)
			if err := creditRows.Scan(&id, &rawNarration, &cleanedPayee, &payMode, &amount,
				&accID, &bankName, &accType, &mask, &nick, &variant); err == nil {

				totalInflow += amount
				accInflow[accID] += amount

				displayName := bankName
				if nick != "" {
					displayName = nick
				} else if variant != "" {
					displayName = fmt.Sprintf("%s %s", bankName, variant)
				} else if mask != "" {
					displayName = fmt.Sprintf("%s (%s)", bankName, mask)
				}
				accNames[accID] = displayName
				accTypes[accID] = accType

				// Classify Source
				upperNarration := strings.ToUpper(rawNarration)
				upperPayee := strings.ToUpper(cleanedPayee)
				var srcID string

				if payMode == "SALARY" || strings.Contains(upperNarration, "SALARY") ||
					strings.Contains(upperNarration, "PAYROLL") || strings.Contains(upperNarration, "SAL/") ||
					strings.Contains(upperNarration, "MONTHLY SAL") || strings.Contains(upperPayee, "SALARY") {
					srcID = "src_salary"
				} else if strings.Contains(upperNarration, "REFUND") || strings.Contains(upperNarration, "CASHBACK") ||
					strings.Contains(upperNarration, "REVERSAL") || strings.Contains(upperNarration, "SWIGGY REFUND") ||
					strings.Contains(upperNarration, "AMAZON REFUND") {
					srcID = "src_refund"
				} else if strings.Contains(upperNarration, "INTEREST") || strings.Contains(upperNarration, "DIVIDEND") ||
					strings.Contains(upperNarration, "ZERODHA") || strings.Contains(upperNarration, "GROWW") ||
					strings.Contains(upperNarration, "MUTUAL FUND") || strings.Contains(upperNarration, "FD ") ||
					strings.Contains(upperNarration, "TERM DEP") {
					srcID = "src_invest"
				} else {
					srcID = "src_other"
				}

				sourceTotals[srcID] += amount
				sourceCounts[srcID]++

				if sourceToAcc[srcID] == nil {
					sourceToAcc[srcID] = make(map[string]float64)
				}
				sourceToAcc[srcID][accID] += amount
			}
		}
		creditRows.Close()
	}

	// 5. Extract Outflows and Categories (Layer 2 & Layer 3)
	outflowQuery := fmt.Sprintf(`
		SELECT 
			t.id, t.amount, t.payment_mode, t.cleaned_payee,
			a.id as acc_id, a.bank_name, a.account_type, COALESCE(a.account_number_mask, '') as mask,
			COALESCE(a.nickname, '') as nickname, COALESCE(a.card_variant, '') as variant,
			COALESCE(c.id, 'cat_others') as cat_id,
			COALESCE(c.name, 'Others & Uncategorized') as cat_name,
			COALESCE(c.color_hex, '#94A3B8') as cat_color,
			COALESCE(c.icon, 'HelpCircle') as cat_icon
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.tx_type = 'DEBIT' AND t.is_transfer = 0 AND t.is_excluded = 0 AND %s
	`, dateClause)

	debitRows, err := d.conn.Query(outflowQuery, dateArgs...)
	categoryTotals := make(map[string]float64)     // cat_id -> amount
	categoryCounts := make(map[string]int)         // cat_id -> count
	categoryMeta := make(map[string]struct {
		name, color, icon string
	})

	cardSpends := make(map[string]float64)         // card_acc_id -> amount
	cardToCat := make(map[string]map[string]float64) // card_acc_id -> cat_id -> amount
	bankToCat := make(map[string]map[string]float64) // bank_acc_id -> cat_id -> amount
	accOutflow := make(map[string]float64)         // acc_id -> total outflow

	var totalOutflow float64
	var creditCardSpendTotal float64
	var directBankSpendTotal float64
	var fixedNeedsSpend float64
	var discretionarySpend float64

	fixedCategoryIDs := map[string]bool{
		"cat_groceries": true,
		"cat_bills":     true,
		"cat_travel":    true,
		"cat_health":    true,
		"cat_charges":   true,
	}

	if err == nil {
		for debitRows.Next() {
			var (
				id, payMode, cleanedPayee string
				amount                    float64
				accID, bankName, accType, mask, nick, variant string
				catID, catName, catColor, catIcon string
			)
			if err := debitRows.Scan(&id, &amount, &payMode, &cleanedPayee,
				&accID, &bankName, &accType, &mask, &nick, &variant,
				&catID, &catName, &catColor, &catIcon); err == nil {

				totalOutflow += amount
				categoryTotals[catID] += amount
				categoryCounts[catID]++
				categoryMeta[catID] = struct{ name, color, icon string }{catName, catColor, catIcon}
				accOutflow[accID] += amount

				displayName := bankName
				if nick != "" {
					displayName = nick
				} else if variant != "" {
					displayName = fmt.Sprintf("%s %s", bankName, variant)
				} else if mask != "" {
					displayName = fmt.Sprintf("%s (%s)", bankName, mask)
				}
				accNames[accID] = displayName
				accTypes[accID] = accType

				if fixedCategoryIDs[catID] || strings.Contains(strings.ToLower(catName), "grocer") ||
					strings.Contains(strings.ToLower(catName), "bill") || strings.Contains(strings.ToLower(catName), "health") ||
					strings.Contains(strings.ToLower(catName), "util") {
					fixedNeedsSpend += amount
				} else {
					discretionarySpend += amount
				}

				if accType == "CREDIT_CARD" {
					creditCardSpendTotal += amount
					cardSpends[accID] += amount
					if cardToCat[accID] == nil {
						cardToCat[accID] = make(map[string]float64)
					}
					cardToCat[accID][catID] += amount
				} else {
					directBankSpendTotal += amount
					if bankToCat[accID] == nil {
						bankToCat[accID] = make(map[string]float64)
					}
					bankToCat[accID][catID] += amount
				}
			}
		}
		debitRows.Close()
	}

	// 6. Build Nodes and Links for Sankey
	nodes := []models.SankeyNode{}
	links := []models.SankeyLink{}

	sourceMeta := map[string]struct {
		name, color, icon string
	}{
		"src_salary": {"Salary & Wages", "#10B981", "Briefcase"},
		"src_refund": {"Refunds & Cashbacks", "#06B6D4", "RotateCcw"},
		"src_invest": {"Investments & Returns", "#8B5CF6", "TrendingUp"},
		"src_other":  {"Other Inflows & UPI", "#3B82F6", "ArrowDownLeft"},
	}

	// Layer 0: Income Sources
	for srcID, total := range sourceTotals {
		if total <= 0 {
			continue
		}
		meta := sourceMeta[srcID]
		nodes = append(nodes, models.SankeyNode{
			ID:         srcID,
			Name:       meta.name,
			Type:       models.SankeyNodeIncomeSource,
			ColorHex:   meta.color,
			Icon:       meta.icon,
			TotalValue: total,
			Layer:      0,
			TxCount:    sourceCounts[srcID],
		})

		// Links to Bank Accounts (Layer 0 -> Layer 1)
		for accID, val := range sourceToAcc[srcID] {
			if val > 0 {
				links = append(links, models.SankeyLink{
					Source:   srcID,
					Target:   "acc_" + accID,
					Value:    val,
					ColorHex: meta.color,
				})
			}
		}
	}

	// If Outflow > Inflow, add a Deficit / Prior Savings Source Node to balance Layer 0
	if totalOutflow > totalInflow {
		deficit := totalOutflow - totalInflow
		deficitNodeID := "src_prior_balance"
		nodes = append(nodes, models.SankeyNode{
			ID:         deficitNodeID,
			Name:       "Prior Balance / Deficit",
			Type:       models.SankeyNodeDeficit,
			ColorHex:   "#F59E0B",
			Icon:       "AlertTriangle",
			TotalValue: deficit,
			Layer:      0,
			TxCount:    1,
		})

		// Find the bank accounts or cards that spent beyond income
		for accID := range accNames {
			if accTypes[accID] != "CREDIT_CARD" {
				links = append(links, models.SankeyLink{
					Source:   deficitNodeID,
					Target:   "acc_" + accID,
					Value:    deficit,
					ColorHex: "#F59E0B",
				})
				break
			}
		}
		if len(accNames) == 0 {
			// If no bank account found, create a fallback account node
			fallbackAccID := "acc_main"
			nodes = append(nodes, models.SankeyNode{
				ID:         fallbackAccID,
				Name:       "Primary Account",
				Type:       models.SankeyNodeAccount,
				ColorHex:   "#3B82F6",
				Icon:       "Landmark",
				TotalValue: totalOutflow,
				Layer:      1,
			})
			links = append(links, models.SankeyLink{
				Source:   deficitNodeID,
				Target:   fallbackAccID,
				Value:    deficit,
				ColorHex: "#F59E0B",
			})
		}
	}

	// Layer 1: Bank Accounts (Savings / Current)
	for accID, name := range accNames {
		if accTypes[accID] == "CREDIT_CARD" {
			continue // Handled in Channel layer
		}
		inVal := accInflow[accID]
		outVal := accOutflow[accID]
		val := math.Max(inVal, outVal)
		if val <= 0 {
			continue
		}

		nodes = append(nodes, models.SankeyNode{
			ID:         "acc_" + accID,
			Name:       name,
			Type:       models.SankeyNodeAccount,
			ColorHex:   "#3B82F6",
			Icon:       "Landmark",
			TotalValue: val,
			Layer:      1,
		})

		// Direct Bank -> Categories (Layer 1 -> Layer 3)
		for catID, amt := range bankToCat[accID] {
			if amt > 0 {
				links = append(links, models.SankeyLink{
					Source:   "acc_" + accID,
					Target:   "cat_" + catID,
					Value:    amt,
					ColorHex: "#60A5FA",
				})
			}
		}

		// Bank -> Credit Cards (Layer 1 -> Layer 2)
		for cardAccID, cardAmt := range cardSpends {
			if cardAmt > 0 {
				links = append(links, models.SankeyLink{
					Source:   "acc_" + accID,
					Target:   "chan_card_" + cardAccID,
					Value:    cardAmt,
					ColorHex: "#818CF8",
				})
				break // Link from primary bank
			}
		}
	}

	// Layer 2: Credit Card Channels
	for cardAccID, cardAmt := range cardSpends {
		if cardAmt <= 0 {
			continue
		}
		cardName := accNames[cardAccID]
		if cardName == "" {
			cardName = "Credit Card"
		}
		chanID := "chan_card_" + cardAccID

		nodes = append(nodes, models.SankeyNode{
			ID:         chanID,
			Name:       cardName,
			Type:       models.SankeyNodeChannel,
			ColorHex:   "#6366F1",
			Icon:       "CreditCard",
			TotalValue: cardAmt,
			Layer:      2,
		})

		// Card Channel -> Categories (Layer 2 -> Layer 3)
		for catID, amt := range cardToCat[cardAccID] {
			if amt > 0 {
				links = append(links, models.SankeyLink{
					Source:   chanID,
					Target:   "cat_" + catID,
					Value:    amt,
					ColorHex: "#A5B4FC",
				})
			}
		}
	}

	// Layer 3: Expense Categories
	for catID, total := range categoryTotals {
		if total <= 0 {
			continue
		}
		meta := categoryMeta[catID]
		nodes = append(nodes, models.SankeyNode{
			ID:         "cat_" + catID,
			Name:       meta.name,
			Type:       models.SankeyNodeCategory,
			ColorHex:   meta.color,
			Icon:       meta.icon,
			TotalValue: total,
			Layer:      3,
			TxCount:    categoryCounts[catID],
		})
	}

	// Layer 3: Net Surplus / Savings Node (if positive)
	netSurplus := totalInflow - totalOutflow
	var savingsRate float64
	if totalInflow > 0 {
		savingsRate = (netSurplus / totalInflow) * 100.0
	}

	if netSurplus > 0 {
		surplusNodeID := "dest_surplus"
		nodes = append(nodes, models.SankeyNode{
			ID:         surplusNodeID,
			Name:       "Net Savings / Surplus",
			Type:       models.SankeyNodeSurplus,
			ColorHex:   "#10B981",
			Icon:       "PiggyBank",
			TotalValue: netSurplus,
			Layer:      3,
			TxCount:    1,
		})

		// Link from Bank Accounts -> Net Surplus
		for accID := range accNames {
			if accTypes[accID] != "CREDIT_CARD" {
				links = append(links, models.SankeyLink{
					Source:   "acc_" + accID,
					Target:   surplusNodeID,
					Value:    netSurplus,
					ColorHex: "#10B981",
				})
				break
			}
		}
	}

	res.Sankey = models.SankeyData{
		Nodes:        nodes,
		Links:        links,
		TotalInflow:  totalInflow,
		TotalOutflow: totalOutflow,
		NetSurplus:   netSurplus,
		SavingsRate:  savingsRate,
	}

	// 7. Calculate MoM Category Comparisons & Sparklines
	prevCategorySpends := make(map[string]float64)
	if res.PreviousMonth != "" {
		prevRows, err := d.conn.Query(`
			SELECT COALESCE(category_id, 'cat_others'), SUM(amount)
			FROM transactions
			WHERE tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0
			  AND tx_date >= ? AND tx_date <= ?
			GROUP BY category_id
		`, res.PreviousMonth+"-01", res.PreviousMonth+"-31")
		if err == nil {
			for prevRows.Next() {
				var cid string
				var amt float64
				if err := prevRows.Scan(&cid, &amt); err == nil {
					prevCategorySpends[cid] = amt
				}
			}
			prevRows.Close()
		}
	}

	// 3-Month average & 6-month historical sparkline
	for catID, curSpend := range categoryTotals {
		meta := categoryMeta[catID]
		prevSpend := prevCategorySpends[catID]
		delta := curSpend - prevSpend
		var pctChange float64
		if prevSpend > 0 {
			pctChange = (delta / prevSpend) * 100.0
		} else if curSpend > 0 {
			pctChange = 100.0
		}

		trend := "FLAT"
		if delta > 100 {
			trend = "UP"
		} else if delta < -100 {
			trend = "DOWN"
		}

		// Fetch last 6 months history for sparkline
		historyPoints := []models.MonthlyCategoryDataPoint{}
		hRows, err := d.conn.Query(`
			SELECT strftime('%Y-%m', tx_date) as m, COALESCE(SUM(amount), 0)
			FROM transactions
			WHERE (category_id = ? OR (? = 'cat_others' AND category_id IS NULL))
			  AND tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0
			GROUP BY m
			ORDER BY m DESC
			LIMIT 6
		`, catID, catID)
		var threeMonthSum float64
		var threeMonthCount int
		if err == nil {
			for hRows.Next() {
				var dp models.MonthlyCategoryDataPoint
				if err := hRows.Scan(&dp.Month, &dp.Amount); err == nil {
					historyPoints = append([]models.MonthlyCategoryDataPoint{dp}, historyPoints...)
					if threeMonthCount < 3 {
						threeMonthSum += dp.Amount
						threeMonthCount++
					}
				}
			}
			hRows.Close()
		}
		var threeMonthAvg float64
		if threeMonthCount > 0 {
			threeMonthAvg = threeMonthSum / float64(threeMonthCount)
		}

		// Top payees for this category in period
		topPayees := []models.PayeeSpend{}
		topQuery := fmt.Sprintf(`
			SELECT cleaned_payee, payment_mode, SUM(amount), COUNT(id)
			FROM transactions
			WHERE (category_id = ? OR (? = 'cat_others' AND category_id IS NULL))
			  AND tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 AND %s
			GROUP BY cleaned_payee, payment_mode
			ORDER BY SUM(amount) DESC
			LIMIT 3
		`, dateClause)
		pArgs := append([]interface{}{catID, catID}, dateArgs...)
		pRows, err := d.conn.Query(topQuery, pArgs...)
		if err == nil {
			for pRows.Next() {
				var ps models.PayeeSpend
				if err := pRows.Scan(&ps.Payee, &ps.PaymentMode, &ps.TotalSpent, &ps.TxCount); err == nil && ps.Payee != "" {
					topPayees = append(topPayees, ps)
				}
			}
			pRows.Close()
		}

		res.CategoryComparisons = append(res.CategoryComparisons, models.CategoryComparisonItem{
			CategoryID:       catID,
			CategoryName:     meta.name,
			CategoryColor:    meta.color,
			CategoryIcon:     meta.icon,
			CurrentSpend:     curSpend,
			PreviousSpend:    prevSpend,
			ThreeMonthAvg:    threeMonthAvg,
			DeltaAmount:      delta,
			PercentageChange: pctChange,
			Trend:            trend,
			History:          historyPoints,
			TopPayees:        topPayees,
		})
	}

	// 8. Generate Actionable MoM Anomalies & Shift Insights
	for _, comp := range res.CategoryComparisons {
		// Category Spike Anomaly
		if comp.PreviousSpend > 500 && comp.PercentageChange >= 30.0 && comp.DeltaAmount >= 1000.0 {
			topContributorText := ""
			if len(comp.TopPayees) > 0 {
				topContributorText = fmt.Sprintf("Led by %s (₹%.0f)", comp.TopPayees[0].Payee, comp.TopPayees[0].TotalSpent)
			}
			res.Anomalies = append(res.Anomalies, models.MoMAnomaly{
				ID:               "spike_" + comp.CategoryID,
				Type:             "SPIKE",
				Severity:         models.AnomalySeverityWarning,
				Title:            fmt.Sprintf("%s Spend Surged", comp.CategoryName),
				Description:      fmt.Sprintf("%s spend increased by %.1f%% (+₹%.0f) compared to %s.", comp.CategoryName, comp.PercentageChange, comp.DeltaAmount, res.PreviousMonth),
				CategoryName:     comp.CategoryName,
				CategoryColor:    comp.CategoryColor,
				CurrentAmount:    comp.CurrentSpend,
				PreviousAmount:   comp.PreviousSpend,
				DeltaAmount:      comp.DeltaAmount,
				PercentageChange: comp.PercentageChange,
				TopContributor:   topContributorText,
			})
		}

		// Category Drop / Savings
		if comp.PreviousSpend >= 2000.0 && comp.PercentageChange <= -30.0 && comp.DeltaAmount <= -1000.0 {
			res.Anomalies = append(res.Anomalies, models.MoMAnomaly{
				ID:               "drop_" + comp.CategoryID,
				Type:             "DROP",
				Severity:         models.AnomalySeveritySuccess,
				Title:            fmt.Sprintf("%s Spend Dropped", comp.CategoryName),
				Description:      fmt.Sprintf("Great job! %s spend reduced by %.1f%% (-₹%.0f) vs %s.", comp.CategoryName, math.Abs(comp.PercentageChange), math.Abs(comp.DeltaAmount), res.PreviousMonth),
				CategoryName:     comp.CategoryName,
				CategoryColor:    comp.CategoryColor,
				CurrentAmount:    comp.CurrentSpend,
				PreviousAmount:   comp.PreviousSpend,
				DeltaAmount:      comp.DeltaAmount,
				PercentageChange: comp.PercentageChange,
			})
		}

		// New Category Outflow
		if comp.PreviousSpend == 0 && comp.CurrentSpend >= 1500.0 {
			res.Anomalies = append(res.Anomalies, models.MoMAnomaly{
				ID:               "new_" + comp.CategoryID,
				Type:             "NEW_SPEND",
				Severity:         models.AnomalySeverityInfo,
				Title:            fmt.Sprintf("New Expense in %s", comp.CategoryName),
				Description:      fmt.Sprintf("Recorded ₹%.0f in %s with no previous spend in %s.", comp.CurrentSpend, comp.CategoryName, res.PreviousMonth),
				CategoryName:     comp.CategoryName,
				CategoryColor:    comp.CategoryColor,
				CurrentAmount:    comp.CurrentSpend,
				PreviousAmount:   0,
				DeltaAmount:      comp.CurrentSpend,
				PercentageChange: 100.0,
			})
		}
	}

	// Deficit vs Savings Milestone
	if totalInflow > 0 && totalOutflow > totalInflow {
		res.Anomalies = append(res.Anomalies, models.MoMAnomaly{
			ID:               "overall_deficit",
			Type:             "DEFICIT",
			Severity:         models.AnomalySeverityDanger,
			Title:            "Monthly Spend Exceeded Income",
			Description:      fmt.Sprintf("Total outflow (₹%.0f) exceeded total income (₹%.0f) by ₹%.0f this period.", totalOutflow, totalInflow, totalOutflow-totalInflow),
			CurrentAmount:    totalOutflow,
			PreviousAmount:   totalInflow,
			DeltaAmount:      totalOutflow - totalInflow,
			PercentageChange: ((totalOutflow - totalInflow) / totalInflow) * 100.0,
		})
	} else if totalInflow > 0 && savingsRate >= 40.0 {
		res.Anomalies = append(res.Anomalies, models.MoMAnomaly{
			ID:               "high_savings",
			Type:             "SAVINGS_MILESTONE",
			Severity:         models.AnomalySeveritySuccess,
			Title:            "Exceptional Savings Milestone",
			Description:      fmt.Sprintf("You retained %.1f%% of your total income as net surplus (₹%.0f) this period!", savingsRate, netSurplus),
			CurrentAmount:    netSurplus,
			PercentageChange: savingsRate,
		})
	}

	// 9. Find Top Spending Category
	var topCategory string
	var topCategoryAmount float64
	for _, comp := range res.CategoryComparisons {
		if comp.CurrentSpend > topCategoryAmount {
			topCategoryAmount = comp.CurrentSpend
			topCategory = comp.CategoryName
		}
	}

	var ccSharePct, bankSharePct float64
	if totalOutflow > 0 {
		ccSharePct = (creditCardSpendTotal / totalOutflow) * 100.0
		bankSharePct = (directBankSpendTotal / totalOutflow) * 100.0
	}

	res.Summary = models.CashFlowIntelligenceSummary{
		TotalInflow:         totalInflow,
		TotalOutflow:        totalOutflow,
		NetSurplus:          netSurplus,
		SavingsRate:         savingsRate,
		FixedNeedsSpend:     fixedNeedsSpend,
		DiscretionarySpend:  discretionarySpend,
		CreditCardSharePct:  ccSharePct,
		DirectBankSharePct:  bankSharePct,
		TopSpendingCategory: topCategory,
		TopSpendingAmount:   topCategoryAmount,
	}

	return res, nil
}

// -------------------------------------------------------------
// Financial Wrapped / Year in Review Engine
// -------------------------------------------------------------

func (d *DB) GetWrappedStory(year string) (*models.WrappedStory, error) {
	story := &models.WrappedStory{
		AvailableYears: []string{},
		TopMerchants:   []models.WrappedMerchantHighlight{},
		TopCategories:  []models.CategorySpend{},
	}

	// 1. Fetch available years
	yRows, err := d.conn.Query(`
		SELECT DISTINCT strftime('%Y', tx_date) as y
		FROM transactions
		WHERE tx_date IS NOT NULL AND tx_date != ''
		ORDER BY y DESC
	`)
	if err == nil {
		for yRows.Next() {
			var y string
			if err := yRows.Scan(&y); err == nil && y != "" {
				story.AvailableYears = append(story.AvailableYears, y)
			}
		}
		yRows.Close()
	}

	selectedYear := strings.TrimSpace(year)
	if selectedYear == "" || selectedYear == "latest" {
		if len(story.AvailableYears) > 0 {
			selectedYear = story.AvailableYears[0]
		} else {
			selectedYear = time.Now().Format("2006")
		}
	}
	story.Year = selectedYear

	var dateFilter string
	var dateArgs []interface{}
	if selectedYear == "ALL" {
		dateFilter = "1=1"
	} else {
		dateFilter = "tx_date >= ? AND tx_date <= ?"
		dateArgs = append(dateArgs, selectedYear+"-01-01", selectedYear+"-12-31")
	}

	// 2. Aggregate stats
	statsQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(CASE WHEN tx_type = 'CREDIT' AND is_transfer = 0 AND is_excluded = 0 THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 THEN amount ELSE 0 END), 0) as expense,
			COUNT(CASE WHEN is_transfer = 0 AND is_excluded = 0 THEN id ELSE NULL END) as total_txs,
			COUNT(CASE WHEN payment_mode = 'UPI' AND is_transfer = 0 AND is_excluded = 0 THEN id ELSE NULL END) as upi_txs,
			COUNT(CASE WHEN payment_mode IN ('CARD_POS', 'CARD_ONLINE') AND is_transfer = 0 AND is_excluded = 0 THEN id ELSE NULL END) as card_txs,
			COALESCE(SUM(CASE WHEN payment_mode = 'UPI' AND tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 THEN amount ELSE 0 END), 0) as upi_spend,
			COALESCE(SUM(CASE WHEN payment_mode IN ('CARD_POS', 'CARD_ONLINE') AND tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 THEN amount ELSE 0 END), 0) as card_spend,
			COALESCE(SUM(cashback_amount), 0) as cashback,
			COALESCE(SUM(reward_points_earned), 0) as reward_pts
		FROM transactions
		WHERE %s
	`, dateFilter)

	row := d.conn.QueryRow(statsQuery, dateArgs...)
	_ = row.Scan(
		&story.TotalIncome,
		&story.TotalExpense,
		&story.TotalTransactions,
		&story.UPITxCount,
		&story.CardTxCount,
		&story.TotalUPISpend,
		&story.TotalCardSpend,
		&story.TotalCashback,
		&story.TotalRewardPoints,
	)

	story.NetSavings = story.TotalIncome - story.TotalExpense
	if story.TotalIncome > 0 {
		story.SavingsRate = (story.NetSavings / story.TotalIncome) * 100.0
	}

	// 3. Top Merchants
	merchQuery := fmt.Sprintf(`
		SELECT cleaned_payee, SUM(amount) as spent, COUNT(id) as cnt, payment_mode
		FROM transactions
		WHERE tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 AND %s
		  AND cleaned_payee != '' AND cleaned_payee != 'ATM Cash Withdrawal'
		GROUP BY cleaned_payee
		ORDER BY cnt DESC, spent DESC
		LIMIT 5
	`, dateFilter)

	mRows, err := d.conn.Query(merchQuery, dateArgs...)
	if err == nil {
		for mRows.Next() {
			var m models.WrappedMerchantHighlight
			if err := mRows.Scan(&m.Payee, &m.TotalSpent, &m.OrderCount, &m.PaymentMode); err == nil {
				story.TopMerchants = append(story.TopMerchants, m)
			}
		}
		mRows.Close()
	}
	if len(story.TopMerchants) > 0 {
		story.CrownMerchant = &story.TopMerchants[0]
	}

	// 4. Biggest Single Purchase
	bigQuery := fmt.Sprintf(`
		SELECT id, account_id, tx_hash, tx_date, raw_narration, cleaned_payee, payment_mode, reference_number, tx_type, amount
		FROM transactions
		WHERE tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 AND %s
		ORDER BY amount DESC
		LIMIT 1
	`, dateFilter)
	bigRow := d.conn.QueryRow(bigQuery, dateArgs...)
	var b models.Transaction
	if err := bigRow.Scan(
		&b.ID, &b.AccountID, &b.TxHash, &b.TxDate, &b.RawNarration,
		&b.CleanedPayee, &b.PaymentMode, &b.ReferenceNumber, &b.TxType, &b.Amount,
	); err == nil {
		story.BiggestPurchase = &b
	}

	// 5. Busiest Day of Year
	busyQuery := fmt.Sprintf(`
		SELECT tx_date, SUM(amount), COUNT(id)
		FROM transactions
		WHERE tx_type = 'DEBIT' AND is_transfer = 0 AND is_excluded = 0 AND %s
		GROUP BY tx_date
		ORDER BY SUM(amount) DESC
		LIMIT 1
	`, dateFilter)
	busyRow := d.conn.QueryRow(busyQuery, dateArgs...)
	_ = busyRow.Scan(&story.BusiestDay, &story.BusiestDaySpend, &story.BusiestDayTxCount)

	// 6. Top Categories
	catQuery := fmt.Sprintf(`
		SELECT c.id, c.name, c.color_hex, c.icon, SUM(t.amount) as amt, COUNT(t.id) as cnt
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.tx_type = 'DEBIT' AND t.is_transfer = 0 AND t.is_excluded = 0 AND %s
		GROUP BY c.id
		ORDER BY amt DESC
		LIMIT 5
	`, dateFilter)
	cRows, err := d.conn.Query(catQuery, dateArgs...)
	if err == nil {
		for cRows.Next() {
			var cs models.CategorySpend
			if err := cRows.Scan(&cs.CategoryID, &cs.CategoryName, &cs.ColorHex, &cs.Icon, &cs.TotalAmount, &cs.TxCount); err == nil {
				if story.TotalExpense > 0 {
					cs.Percentage = (cs.TotalAmount / story.TotalExpense) * 100.0
				}
				story.TopCategories = append(story.TopCategories, cs)
			}
		}
		cRows.Close()
	}

	// 7. Persona Archetype Synthesis
	topCatName := ""
	if len(story.TopCategories) > 0 {
		topCatName = strings.ToLower(story.TopCategories[0].CategoryName)
	}

	if story.SavingsRate >= 40.0 && story.TotalIncome > 0 {
		story.PersonaTitle = "The Wealth Architect"
		story.PersonaBadge = "🧘 High-Velocity Capital Builder"
		story.PersonaDescription = "You preserved over 40% of everything you made this year. Your financial runway and capital compounding velocity are in elite territory."
	} else if story.TotalCardSpend > story.TotalUPISpend && (story.TotalCashback >= 1500 || story.TotalRewardPoints >= 5000) {
		story.PersonaTitle = "The Points & Float Tactician"
		story.PersonaBadge = "💳 50-Day Float & Rewards Maestro"
		story.PersonaDescription = "You made credit card algorithms work for you! Maximizing interest-free cash flow and extracting real cashbacks on everyday expenses."
	} else if strings.Contains(topCatName, "food") || strings.Contains(topCatName, "dining") || strings.Contains(topCatName, "grocer") {
		story.PersonaTitle = "The Epicurean Foodie"
		story.PersonaBadge = "🍕 Culinary & Quick-Commerce Maven"
		story.PersonaDescription = "Life is meant to be savored! Dining, gourmet treats, and instant doorstep deliveries defined your favorite lifestyle investments."
	} else if strings.Contains(topCatName, "travel") || strings.Contains(topCatName, "flight") {
		story.PersonaTitle = "The Globetrotter"
		story.PersonaBadge = "✈️ Jetsetter & Experience Collector"
		story.PersonaDescription = "Collecting passport stamps, weekend getaways, and airline miles was your hallmark. Memories over things."
	} else if strings.Contains(topCatName, "shop") {
		story.PersonaTitle = "The Quality Curator"
		story.PersonaBadge = "🛍️ Selective Lifestyle Collector"
		story.PersonaDescription = "Upgrading your daily gear, wardrobe, and lifestyle with deliberate, high-quality selections."
	} else {
		story.PersonaTitle = "The Balanced Pragmatist"
		story.PersonaBadge = "⚡ Systematic Financial Navigator"
		story.PersonaDescription = "A steady hand on the tiller. You maintained consistent cash flows, avoided lifestyle creep, and kept financial peace of mind."
	}

	return story, nil
}

// ==========================================
// SECURITY & LOCAL AUTH OPERATIONS
// ==========================================

// GetDBFilePermissions returns the file permission mode of the SQLite file
func (d *DB) GetDBFilePermissions() string {
	info, err := os.Stat(d.path)
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%v (%04o)", info.Mode().Perm(), info.Mode().Perm())
}

// GetSecuritySettings retrieves current local authentication settings
func (d *DB) GetSecuritySettings() (*models.SecuritySettings, string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var authEnabled bool
	var passwordHash string
	var autoLockMinutes int
	var updatedAt time.Time

	err := d.conn.QueryRow(`
		SELECT auth_enabled, password_hash, auto_lock_minutes, updated_at
		FROM app_security
		WHERE id = 1
	`).Scan(&authEnabled, &passwordHash, &autoLockMinutes, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.SecuritySettings{
				AuthEnabled:     false,
				AutoLockMinutes: 60,
				HasPassword:     false,
				FilePermissions: d.GetDBFilePermissions(),
			}, "", nil
		}
		return nil, "", err
	}

	return &models.SecuritySettings{
		AuthEnabled:     authEnabled,
		AutoLockMinutes: autoLockMinutes,
		HasPassword:     len(passwordHash) > 0,
		FilePermissions: d.GetDBFilePermissions(),
		UpdatedAt:       updatedAt,
	}, passwordHash, nil
}

// SetSecurityPassword configures the master password and enables local auth
func (d *DB) SetSecurityPassword(passwordHash string, autoLockMinutes int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if autoLockMinutes <= 0 {
		autoLockMinutes = 60
	}

	_, err := d.conn.Exec(`
		INSERT INTO app_security (id, auth_enabled, password_hash, auto_lock_minutes, updated_at)
		VALUES (1, 1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			auth_enabled = 1,
			password_hash = excluded.password_hash,
			auto_lock_minutes = excluded.auto_lock_minutes,
			updated_at = CURRENT_TIMESTAMP
	`, passwordHash, autoLockMinutes)
	return err
}

// DisableSecurity disables local authentication
func (d *DB) DisableSecurity() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.conn.Exec(`
		UPDATE app_security
		SET auth_enabled = 0, password_hash = '', updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	return err
}

// UpdateSecuritySettings updates timeout settings
func (d *DB) UpdateSecuritySettings(autoLockMinutes int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if autoLockMinutes <= 0 {
		autoLockMinutes = 60
	}

	_, err := d.conn.Exec(`
		UPDATE app_security
		SET auto_lock_minutes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, autoLockMinutes)
	return err
}

// GetSalaryInsights aggregates career salary earnings, yearly progression, monthly trajectory, employer breakdown, and paychecks
func (d *DB) GetSalaryInsights() (*models.SalaryInsightsResponse, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	res := &models.SalaryInsightsResponse{
		YearlyProgress:  []models.SalaryYearSummary{},
		MonthlyHistory:  []models.SalaryMonthlyDataPoint{},
		Employers:       []models.EmployerSummary{},
		RecentPaychecks: []models.Transaction{},
	}

	query := `
		SELECT 
			t.id, t.account_id, a.bank_name || ' (' || a.account_type || ')' as account_name,
			t.statement_import_id, t.tx_hash, t.tx_date, t.value_date,
			t.raw_narration, t.cleaned_payee, t.payment_mode, t.reference_number,
			t.tx_type, t.amount, t.running_balance,
			t.category_id, c.name, c.color_hex, c.icon,
			t.upi_vpa, t.card_last4, t.merchant_category, t.cashback_amount, t.reward_points_earned,
			t.is_transfer, t.is_excluded, t.transfer_peer_id, t.transfer_match_reason, t.net_amount,
			t.original_currency, t.original_amount,
			t.is_recurring, t.notes, t.tags, t.created_at
		FROM transactions t
		LEFT JOIN accounts a ON t.account_id = a.id
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.tx_type = 'CREDIT' 
		  AND t.is_excluded = 0
		  AND (
		      t.category_id = 'cat_salary' 
		      OR t.payment_mode = 'SALARY'
		      OR UPPER(t.cleaned_payee) LIKE '%SALARY%'
		      OR UPPER(t.cleaned_payee) LIKE '%PAYROLL%'
		      OR UPPER(t.raw_narration) LIKE '%SALARY%'
		      OR UPPER(t.raw_narration) LIKE '%SAL CREDIT%'
		      OR UPPER(t.raw_narration) LIKE '%PAYROLL%'
		      OR UPPER(t.raw_narration) LIKE '%MONTHLY SAL%'
		  )
		ORDER BY t.tx_date ASC, t.id ASC
	`
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query salary transactions: %w", err)
	}
	defer rows.Close()

	var allTxs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var stmtID, valDate, catID, catName, catColor, catIcon, upiVpa, cardLast4, merchCat, peerID, matchReason, origCurr sql.NullString
		var runBal, netAmt, origAmt sql.NullFloat64

		if err := rows.Scan(
			&t.ID, &t.AccountID, &t.AccountName,
			&stmtID, &t.TxHash, &t.TxDate, &valDate,
			&t.RawNarration, &t.CleanedPayee, &t.PaymentMode, &t.ReferenceNumber,
			&t.TxType, &t.Amount, &runBal,
			&catID, &catName, &catColor, &catIcon,
			&upiVpa, &cardLast4, &merchCat, &t.CashbackAmount, &t.RewardPointsEarned,
			&t.IsTransfer, &t.IsExcluded, &peerID, &matchReason, &netAmt,
			&origCurr, &origAmt,
			&t.IsRecurring, &t.Notes, &t.Tags, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan salary transaction: %w", err)
		}

		if stmtID.Valid {
			t.StatementImportID = &stmtID.String
		}
		if valDate.Valid {
			t.ValueDate = &valDate.String
		}
		if runBal.Valid {
			val := runBal.Float64
			t.RunningBalance = &val
		}
		if catID.Valid {
			t.CategoryID = &catID.String
		}
		if catName.Valid {
			t.CategoryName = &catName.String
		}
		if catColor.Valid {
			t.CategoryColor = &catColor.String
		}
		if catIcon.Valid {
			t.CategoryIcon = &catIcon.String
		}
		if upiVpa.Valid {
			t.UPIVPA = &upiVpa.String
		}
		if cardLast4.Valid {
			t.CardLast4 = &cardLast4.String
		}
		if merchCat.Valid {
			t.MerchantCategory = &merchCat.String
		}
		if peerID.Valid {
			t.TransferPeerID = &peerID.String
		}
		if matchReason.Valid {
			t.TransferMatchReason = &matchReason.String
		}
		if netAmt.Valid {
			val := netAmt.Float64
			t.NetAmount = &val
		}
		if origCurr.Valid {
			t.OriginalCurrency = &origCurr.String
		}
		if origAmt.Valid {
			val := origAmt.Float64
			t.OriginalAmount = &val
		}

		allTxs = append(allTxs, t)
	}

	if len(allTxs) == 0 {
		return res, nil
	}

	res.TotalPaychecks = len(allTxs)
	res.FirstSalaryAmount = allTxs[0].Amount
	res.FirstSalaryDate = allTxs[0].TxDate
	res.LatestSalaryAmount = allTxs[len(allTxs)-1].Amount
	res.LatestSalaryDate = allTxs[len(allTxs)-1].TxDate
	res.LatestEmployer = allTxs[len(allTxs)-1].CleanedPayee
	if res.LatestEmployer == "" {
		res.LatestEmployer = allTxs[len(allTxs)-1].RawNarration
	}

	now := time.Now()
	currentYear := now.Year()
	currentFYStart := fmt.Sprintf("%d-04-01", currentYear)
	if now.Month() < time.April {
		currentFYStart = fmt.Sprintf("%d-04-01", currentYear-1)
	}

	yearlyMap := make(map[int]*models.SalaryYearSummary)
	var yearsOrder []int

	monthlyMap := make(map[string]*models.SalaryMonthlyDataPoint)
	var monthsOrder []string

	employerMap := make(map[string]*models.EmployerSummary)
	var employerOrder []string

	var peakAmt float64
	var peakDate, peakEmp string

	for _, tx := range allTxs {
		res.LifetimeEarned += tx.Amount

		if tx.Amount > peakAmt {
			peakAmt = tx.Amount
			peakDate = tx.TxDate
			peakEmp = tx.CleanedPayee
			if peakEmp == "" {
				peakEmp = tx.RawNarration
			}
		}

		// Current Year / Current FY
		if strings.HasPrefix(tx.TxDate, fmt.Sprintf("%d-", currentYear)) {
			res.CurrentYearEarned += tx.Amount
		}
		if tx.TxDate >= currentFYStart {
			res.CurrentFYEarned += tx.Amount
		}

		// Yearly aggregation
		if len(tx.TxDate) >= 4 {
			var y int
			fmt.Sscanf(tx.TxDate[:4], "%d", &y)
			if y > 0 {
				if _, exists := yearlyMap[y]; !exists {
					yearlyMap[y] = &models.SalaryYearSummary{Year: y}
					yearsOrder = append(yearsOrder, y)
				}
				yearlyMap[y].TotalEarned += tx.Amount
				yearlyMap[y].PaycheckCount++
			}
		}

		// Monthly aggregation
		if len(tx.TxDate) >= 7 {
			m := tx.TxDate[:7] // "YYYY-MM"
			emp := tx.CleanedPayee
			if emp == "" {
				emp = tx.RawNarration
			}
			if _, exists := monthlyMap[m]; !exists {
				monthlyMap[m] = &models.SalaryMonthlyDataPoint{
					Month:    m,
					Amount:   tx.Amount,
					Employer: emp,
				}
				monthsOrder = append(monthsOrder, m)
			} else {
				monthlyMap[m].Amount += tx.Amount
			}
		}

		// Employer aggregation
		emp := tx.CleanedPayee
		if emp == "" {
			emp = tx.RawNarration
		}
		emp = strings.TrimSpace(emp)
		if emp == "" {
			emp = "Direct Employer / Payroll"
		}

		if _, exists := employerMap[emp]; !exists {
			employerMap[emp] = &models.EmployerSummary{
				EmployerName:  emp,
				TotalEarned:   tx.Amount,
				FirstPaycheck: tx.TxDate,
				LastPaycheck:  tx.TxDate,
				PaycheckCount: 1,
			}
			employerOrder = append(employerOrder, emp)
		} else {
			employerMap[emp].TotalEarned += tx.Amount
			employerMap[emp].LastPaycheck = tx.TxDate
			employerMap[emp].PaycheckCount++
		}
	}

	res.PeakSalaryAmount = peakAmt
	res.PeakSalaryDate = peakDate
	res.PeakEmployer = peakEmp

	if res.FirstSalaryAmount > 0 {
		res.OverallGrowthPct = ((res.LatestSalaryAmount - res.FirstSalaryAmount) / res.FirstSalaryAmount) * 100.0
	}

	// Sort years ascending
	sort.Ints(yearsOrder)
	var prevYearEarned float64
	for _, y := range yearsOrder {
		ys := yearlyMap[y]
		if ys.PaycheckCount > 0 {
			monthsInYear := float64(ys.PaycheckCount)
			if monthsInYear > 12 {
				monthsInYear = 12
			}
			ys.MonthlyAverage = ys.TotalEarned / monthsInYear
		}
		if prevYearEarned > 0 {
			ys.YoYGrowthPct = ((ys.TotalEarned - prevYearEarned) / prevYearEarned) * 100.0
		}
		prevYearEarned = ys.TotalEarned
		res.YearlyProgress = append(res.YearlyProgress, *ys)
	}

	// Calculate Average Monthly Salary across unique earning months
	if len(monthsOrder) > 0 {
		res.AverageMonthly = res.LifetimeEarned / float64(len(monthsOrder))
	}

	// Sort months ascending and detect hikes / bonuses
	sort.Strings(monthsOrder)
	var prevMonthAmt float64
	for _, m := range monthsOrder {
		mp := monthlyMap[m]
		if prevMonthAmt > 0 && mp.Amount >= prevMonthAmt*1.05 {
			hikePct := ((mp.Amount - prevMonthAmt) / prevMonthAmt) * 100.0
			mp.IsHike = true
			mp.HikePct = hikePct
		}
		if res.AverageMonthly > 0 && mp.Amount >= res.AverageMonthly*1.8 {
			mp.IsBonus = true
		}
		prevMonthAmt = mp.Amount
		res.MonthlyHistory = append(res.MonthlyHistory, *mp)
	}

	// Finalize employers list (sort by TotalEarned DESC)
	for _, empName := range employerOrder {
		es := employerMap[empName]
		if es.PaycheckCount > 0 {
			es.MonthlyAverage = es.TotalEarned / float64(es.PaycheckCount)
		}
		res.Employers = append(res.Employers, *es)
	}
	sort.Slice(res.Employers, func(i, j int) bool {
		return res.Employers[i].TotalEarned > res.Employers[j].TotalEarned
	})

	// Paychecks in reverse chronological order
	for i := len(allTxs) - 1; i >= 0; i-- {
		res.RecentPaychecks = append(res.RecentPaychecks, allTxs[i])
	}

	return res, nil
}
