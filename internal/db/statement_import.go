package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"local-finance/internal/models"
)

// Shared operations preserve identity and user edits in single writes and imports.
type statementConnection interface {
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
	Exec(string, ...any) (sql.Result, error)
	Prepare(string) (*sql.Stmt, error)
}
type statementStore struct{ conn statementConnection }

// StatementWriter is valid only inside WithStatementImport's callback.
type StatementWriter struct{ *statementStore }

func (d *DB) WithStatementImport(write func(*StatementWriter) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := write(&StatementWriter{&statementStore{conn: tx}}); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *statementStore) ListAccounts() ([]models.Account, error) {
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
	return list, rows.Err()
}

func (d *statementStore) GetOrCreateAccount(bankName string, accType models.AccountType, accNum, mask string, custID, ifsc, branch, cardNet, cardVar, accHolder string, creditLimit *float64) (*models.Account, error) {
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
			if _, err := d.conn.Exec(`UPDATE accounts SET account_number = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, accNum, a.ID); err != nil {
				return nil, err
			}
			a.AccountNumber = &accNum
		}
		if mask != "" && a.AccountNumberMask == "" {
			if _, err := d.conn.Exec(`UPDATE accounts SET account_number_mask = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, mask, a.ID); err != nil {
				return nil, err
			}
			a.AccountNumberMask = mask
		}
		if cardNet != "" && a.CardNetwork == nil {
			if _, err := d.conn.Exec(`UPDATE accounts SET card_network = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, cardNet, a.ID); err != nil {
				return nil, err
			}
			a.CardNetwork = &cardNet
		}
		if cardVar != "" && a.CardVariant == nil {
			if _, err := d.conn.Exec(`UPDATE accounts SET card_variant = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, cardVar, a.ID); err != nil {
				return nil, err
			}
			a.CardVariant = &cardVar
		}
		if accHolder != "" && (a.AccountHolderName == nil || *a.AccountHolderName == "") {
			if _, err := d.conn.Exec(`UPDATE accounts SET account_holder_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, accHolder, a.ID); err != nil {
				return nil, err
			}
			a.AccountHolderName = &accHolder
		}
		if creditLimit != nil && a.CreditLimit == nil {
			if _, err := d.conn.Exec(`UPDATE accounts SET credit_limit = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *creditLimit, a.ID); err != nil {
				return nil, err
			}
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
		if err := d.SeedDefaultCardRulesIfEmpty(res.ID, cardVar, bankName); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (d *statementStore) CreateStatementImport(s *models.StatementImport) error {
	_, err := d.conn.Exec(`
		INSERT INTO statement_imports (
			id, account_id, filename, file_hash, statement_format, parser_used,
			start_date, end_date, total_transactions, opening_balance, closing_balance, total_debits, total_credits, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.AccountID, s.Filename, s.FileHash, s.StatementFormat, s.ParserUsed,
		s.StartDate, s.EndDate, s.TotalTransactions, s.OpeningBalance, s.ClosingBalance, s.TotalDebits, s.TotalCredits, s.ImportedAt)
	return err
}

func (d *statementStore) UpsertTransaction(tx *models.Transaction) (bool, error) {
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

func (d *statementStore) RecalculateAccountBalance(accountID string) error {
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

func (d *statementStore) ListRules() ([]models.CategorizationRule, error) {
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
	return list, rows.Err()
}

func (d *statementStore) CreateOrUpdateCreditCardBill(b *models.CreditCardBill) error {
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

func (d *statementStore) SeedDefaultCardRulesIfEmpty(accountID string, variant string, bank string) error {
	var count int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM card_reward_rules WHERE account_id = ?`, accountID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	v := strings.ToUpper(variant + " " + bank)
	var defaultRules []models.CardRewardRule

	if strings.Contains(v, "SWIGGY") {
		if _, err := d.conn.Exec(`UPDATE accounts SET card_color = '#EA580C', reward_type = 'CASHBACK', base_reward_rate = 1.0, annual_fee = 500, fee_waiver_threshold = 200000, billing_day = 12 WHERE id = ?`, accountID); err != nil {
			return err
		}
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "SWIGGY|DINEOUT|INSTAMART|GENIE", CategoryName: "Food & Dining", RewardPercentage: 10.0, RewardDescription: "10% Cashback on Swiggy, Food Delivery, Instamart & Dineout", MaxCapPerMonth: floatPtr(1500)},
			{MerchantPattern: "AMAZON|FLIPKART|MYNTRA|BLINKIT|ZEPTO|UBER|OLA|NYKAA|BIGBASKET|CULT.FIT|BOOKMYSHOW", CategoryName: "Online Shopping", RewardPercentage: 5.0, RewardDescription: "5% Cashback on top E-Commerce & quick commerce platforms", MaxCapPerMonth: floatPtr(1500)},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.0, RewardDescription: "1% Unlimited Cashback on all other retail spends"},
		}
	} else if strings.Contains(v, "AMAZON") || (strings.Contains(v, "ICICI") && strings.Contains(v, "PAY")) {
		if _, err := d.conn.Exec(`UPDATE accounts SET card_color = '#1E293B', reward_type = 'CASHBACK', base_reward_rate = 1.0, annual_fee = 0, fee_waiver_threshold = 0, billing_day = 13 WHERE id = ?`, accountID); err != nil {
			return err
		}
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "AMAZON|AMAZON.IN|AMAZON PAY", CategoryName: "Shopping", RewardPercentage: 5.0, RewardDescription: "5% Unlimited Cashback on Amazon Shopping for Prime members"},
			{MerchantPattern: "RECHARGE|BILL|ELECTRICITY|GAS|DTH|POSTPAID|BROADBAND|INSURANCE|SWIGGY|ZOMATO|UBER", CategoryName: "Utilities & Payments", RewardPercentage: 2.0, RewardDescription: "2% Unlimited Cashback on Amazon Pay utilities & 100+ partner merchants"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.0, RewardDescription: "1% Unlimited Cashback on all other retail & dining spends"},
		}
	} else if strings.Contains(v, "FLIPKART") || strings.Contains(v, "AXIS") {
		if _, err := d.conn.Exec(`UPDATE accounts SET card_color = '#0284C7', reward_type = 'CASHBACK', base_reward_rate = 1.5, annual_fee = 500, fee_waiver_threshold = 350000, billing_day = 15 WHERE id = ?`, accountID); err != nil {
			return err
		}
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "FLIPKART|MYNTRA|SHOPSY", CategoryName: "Shopping", RewardPercentage: 5.0, RewardDescription: "5% Unlimited Cashback on Flipkart, Myntra & Cleartrip"},
			{MerchantPattern: "SWIGGY|UBER|PVR|CULT.FIT|TATA 1MG", CategoryName: "Preferred Partners", RewardPercentage: 4.0, RewardDescription: "4% Unlimited Cashback on preferred merchant partners"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "Other Spends", RewardPercentage: 1.5, RewardDescription: "1.5% Unlimited Cashback on all other eligible spends"},
		}
	} else if strings.Contains(v, "REGALIA") || strings.Contains(v, "DINERS") {
		if _, err := d.conn.Exec(`UPDATE accounts SET card_color = '#1E3A8A', reward_type = 'REWARD_POINTS', base_reward_rate = 2.67, annual_fee = 2500, fee_waiver_threshold = 300000, billing_day = 16 WHERE id = ?`, accountID); err != nil {
			return err
		}
		defaultRules = []models.CardRewardRule{
			{MerchantPattern: "SMARTBUY|FLIGHT|HOTEL|CLEARTRIP|YATRA", CategoryName: "Travel & Flights", RewardPercentage: 13.3, RewardDescription: "5X Reward Points (13.3% value) on SmartBuy Flights & Hotels"},
			{MerchantPattern: "DINING|RESTAURANT|ZOMATO|SWIGGY", CategoryName: "Dining", RewardPercentage: 4.0, RewardDescription: "2X Reward Points on Dining spends"},
			{MerchantPattern: "ALL_OTHER", CategoryName: "General Spends", RewardPercentage: 2.67, RewardDescription: "4 Reward Points per ₹150 spent across all categories"},
		}
	} else if strings.Contains(v, "RUPAY") {
		if _, err := d.conn.Exec(`UPDATE accounts SET card_color = '#831843', reward_type = 'CASHBACK', base_reward_rate = 1.0, annual_fee = 250, fee_waiver_threshold = 50000, billing_day = 1 WHERE id = ?`, accountID); err != nil {
			return err
		}
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
		if _, err := d.conn.Exec(`
			INSERT INTO card_reward_rules (
				id, account_id, merchant_pattern, category_name, reward_percentage,
				reward_description, max_cap_per_month, min_spend_per_txn, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, r.ID, r.AccountID, r.MerchantPattern, r.CategoryName, r.RewardPercentage, r.RewardDescription, r.MaxCapPerMonth, r.MinSpendPerTxn, r.CreatedAt, r.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}
