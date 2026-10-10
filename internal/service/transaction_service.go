package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"local-finance/internal/db"
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"local-finance/internal/parser/extractor"
)

type TransactionService struct {
	db       *db.DB
	registry *parser.Registry
}

const MaxStatementFileSize int64 = 20 << 20

func readStatement(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxStatementFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxStatementFileSize {
		return nil, fmt.Errorf("statement exceeds 20 MB")
	}
	return data, nil
}

func NewTransactionService(database *db.DB) *TransactionService {
	return &TransactionService{
		db:       database,
		registry: parser.DefaultRegistry,
	}
}

func (s *TransactionService) ImportStatement(filename string, r io.Reader, manualAccountID string, manualParserID string, password string) (*models.ImportResult, error) {
	fileBytes, err := readStatement(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 1. Calculate File Checksum
	fileHasher := sha256.New()
	fileHasher.Write(fileBytes)
	fileHash := hex.EncodeToString(fileHasher.Sum(nil))

	// If it's a PDF, decrypt first with password if provided so detection can inspect plaintext
	workingBytes := fileBytes
	isPDF := strings.HasSuffix(strings.ToUpper(filename), ".PDF") || (len(fileBytes) > 4 && string(fileBytes[:4]) == "%PDF")
	if isPDF {
		decrypted, decErr := extractor.DecryptPDFIfNeeded(fileBytes, password)
		if decErr == nil && len(decrypted) > 0 {
			workingBytes = decrypted
		} else if decErr != nil {
			if errors.Is(decErr, extractor.ErrPDFPasswordRequired) || password != "" {
				return nil, decErr
			}
		}
	}

	// 2. Select or Auto-detect Parser
	var selectedParser parser.StatementParser
	var confidence float64
	var detectedMeta parser.StatementMeta

	if manualParserID != "" {
		p, exists := s.registry.Get(manualParserID)
		if !exists {
			return nil, fmt.Errorf("specified parser '%s' not found", manualParserID)
		}
		selectedParser = p
		confidence = 1.0
	} else {
		selectedParser, confidence, detectedMeta = s.registry.Detect(filename, workingBytes)
		if selectedParser == nil || confidence < 0.2 {
			return nil, fmt.Errorf("could not auto-detect format for file '%s'. Please choose a specific bank format", filename)
		}
	}

	// 3. Run Parser
	parsedTxList, meta, err := selectedParser.Parse(bytes.NewReader(workingBytes), parser.ParseOptions{
		AccountID: manualAccountID,
		Password:  password,
		Filename:  filename,
	})
	if err != nil {
		return nil, fmt.Errorf("parser failed: %w", err)
	}

	if len(parsedTxList) == 0 {
		return nil, fmt.Errorf("no transactions found in statement. Please verify the file format")
	}

	var account *models.Account
	var stmtImportID string
	var insertedCount, duplicateCount int
	err = s.db.WithStatementImport(func(writer *db.StatementWriter) error {
		// 4. Resolve Account
		account = nil
		if manualAccountID != "" {
			accounts, err := writer.ListAccounts()
			if err != nil {
				return err
			}
			for _, a := range accounts {
				if a.ID == manualAccountID {
					account = &a
					break
				}
			}
		}

		if manualAccountID != "" && account == nil {
			return fmt.Errorf("selected account no longer exists")
		}
		if account == nil {
			bankName := meta.BankName
			if bankName == "" {
				bankName = detectedMeta.BankName
			}
			if bankName == "" {
				bankName = "Imported Bank Account"
			}
			accType := meta.AccountType
			if accType == "" {
				accType = detectedMeta.AccountType
			}
			if accType == "" {
				accType = models.AccountTypeSavings
			}

			var credLimitPtr *float64
			if meta.CreditLimit > 0 {
				credLimitPtr = &meta.CreditLimit
			}

			acc, err := writer.GetOrCreateAccount(bankName, accType, meta.AccountNumber, meta.AccountNumberMask, meta.CustomerID, meta.IFSCCode, meta.BranchName, meta.CardNetwork, meta.CardVariant, meta.AccountHolderName, credLimitPtr)
			if err != nil {
				return fmt.Errorf("failed to get/create account: %w", err)
			}
			account = acc
		}

		// 5. Create Statement Import Log
		stmtImportID = uuid.New().String()
		stmtImport := &models.StatementImport{
			ID:                stmtImportID,
			AccountID:         account.ID,
			Filename:          filename,
			FileHash:          fileHash,
			StatementFormat:   meta.StatementFormat,
			ParserUsed:        selectedParser.ID(),
			StartDate:         &meta.StartDate,
			EndDate:           &meta.EndDate,
			TotalTransactions: len(parsedTxList),
			ImportedAt:        time.Now(),
		}
		if meta.OpeningBalance > 0 {
			stmtImport.OpeningBalance = &meta.OpeningBalance
		}
		if meta.ClosingBalance > 0 {
			stmtImport.ClosingBalance = &meta.ClosingBalance
		}
		if meta.TotalDebits > 0 {
			stmtImport.TotalDebits = &meta.TotalDebits
		}
		if meta.TotalCredits > 0 {
			stmtImport.TotalCredits = &meta.TotalCredits
		}

		if err := writer.CreateStatementImport(stmtImport); err != nil {
			return fmt.Errorf("failed to record statement: %w", err)
		}

		// 6. Record Credit Card Bill if dates present
		stmtDate := meta.StatementDate
		if stmtDate == "" && meta.EndDate != "" {
			stmtDate = meta.EndDate
		}
		if stmtDate != "" && meta.PaymentDueDate != "" {
			var credLimit *float64
			if meta.CreditLimit > 0 {
				credLimit = &meta.CreditLimit
			}
			var availLimit *float64
			if meta.AvailableCreditLimit > 0 {
				availLimit = &meta.AvailableCreditLimit
			}

			if err := writer.CreateOrUpdateCreditCardBill(&models.CreditCardBill{
				AccountID:            account.ID,
				StatementImportID:    &stmtImportID,
				StatementDate:        stmtDate,
				PaymentDueDate:       meta.PaymentDueDate,
				TotalDueAmount:       meta.TotalDueAmount,
				MinimumDueAmount:     &meta.MinimumDueAmount,
				RewardPointsEarned:   meta.RewardPointsEarned,
				RewardPointsBalance:  meta.RewardPointsBalance,
				CashbackEarned:       meta.CashbackEarned,
				CashbackCredited:     meta.CashbackCredited,
				FinanceCharges:       meta.FinanceCharges,
				CreditLimit:          credLimit,
				AvailableCreditLimit: availLimit,
				PaymentStatus:        "UNPAID",
			}); err != nil {
				return fmt.Errorf("failed to record card bill: %w", err)
			}
		}

		// 7. Fetch Categorization Rules for auto-tagging
		rules, err := writer.ListRules()
		if err != nil {
			return err
		}

		// 8. Upsert Transactions
		insertedCount = 0
		duplicateCount = 0

		for _, pt := range parsedTxList {
			// Calculate deterministic fingerprint hash
			txHash := s.calculateTxHash(account.ID, pt.Date, pt.Amount, pt.RawNarration, pt.ReferenceNumber, pt.TxType)

			// Auto-categorize
			catID := s.matchCategory(pt, rules)

			var merchCat *string
			if pt.MerchantCategory != "" {
				merchCat = &pt.MerchantCategory
			}

			tx := &models.Transaction{
				AccountID:          account.ID,
				StatementImportID:  &stmtImportID,
				TxHash:             txHash,
				TxDate:             pt.Date,
				ValueDate:          pt.ValueDate,
				RawNarration:       pt.RawNarration,
				CleanedPayee:       pt.CleanedPayee,
				PaymentMode:        pt.PaymentMode,
				ReferenceNumber:    pt.ReferenceNumber,
				TxType:             pt.TxType,
				Amount:             pt.Amount,
				RunningBalance:     pt.RunningBalance,
				CategoryID:         catID,
				UPIVPA:             pt.UPIVPA,
				CardLast4:          pt.CardLast4,
				MerchantCategory:   merchCat,
				CashbackAmount:     pt.CashbackAmount,
				RewardPointsEarned: pt.RewardPointsEarned,
				IsTransfer:         pt.IsTransfer,
				OriginalCurrency:   pt.OriginalCurrency,
				OriginalAmount:     pt.OriginalAmount,
			}

			isNew, err := writer.UpsertTransaction(tx)
			if err != nil {
				return fmt.Errorf("failed to save transaction: %w", err)
			}
			if isNew {
				insertedCount++
			} else {
				duplicateCount++
			}
		}

		// 9. Recalculate account balance
		if err := writer.RecalculateAccountBalance(account.ID); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 10. Automatically scan and update recurring subscriptions
	subService := NewSubscriptionService(s.db)
	_, _ = subService.ScanAndDetectSubscriptions()

	return &models.ImportResult{
		StatementImportID: stmtImportID,
		AccountID:         account.ID,
		BankName:          account.BankName,
		AccountType:       string(account.AccountType),
		ParserUsed:        selectedParser.Name(),
		Confidence:        confidence,
		TotalParsed:       len(parsedTxList),
		InsertedCount:     insertedCount,
		DuplicateCount:    duplicateCount,
		StartDate:         meta.StartDate,
		EndDate:           meta.EndDate,
	}, nil
}

func (s *TransactionService) PreviewStatement(filename string, r io.Reader, manualAccountID string, manualParserID string, password string) (*models.StatementPreviewResult, error) {
	fileBytes, err := readStatement(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	res := &models.StatementPreviewResult{
		FileID:   uuid.New().String(),
		Filename: filename,
		FileSize: int64(len(fileBytes)),
		Warnings: []string{},
	}

	// 1. Calculate File Checksum
	fileHasher := sha256.New()
	fileHasher.Write(fileBytes)
	res.FileHash = hex.EncodeToString(fileHasher.Sum(nil))

	// Check if PDF and handle decryption
	workingBytes := fileBytes
	isPDF := strings.HasSuffix(strings.ToUpper(filename), ".PDF") || (len(fileBytes) > 4 && string(fileBytes[:4]) == "%PDF")
	if isPDF {
		decrypted, decErr := extractor.DecryptPDFIfNeeded(fileBytes, password)
		if decErr == nil && len(decrypted) > 0 {
			workingBytes = decrypted
		} else if decErr != nil {
			if errors.Is(decErr, extractor.ErrPDFPasswordRequired) {
				res.RequiresPassword = true
				res.Error = decErr.Error()
				return res, nil
			}
			res.Error = fmt.Sprintf("Failed to open PDF: %s", decErr)
			return res, nil
		}
	}

	// 2. Select or Auto-detect Parser
	var selectedParser parser.StatementParser
	var confidence float64
	var detectedMeta parser.StatementMeta

	if manualParserID != "" {
		p, exists := s.registry.Get(manualParserID)
		if !exists {
			res.Error = fmt.Sprintf("specified parser '%s' not found", manualParserID)
			return res, nil
		}
		selectedParser = p
		confidence = 1.0
	} else {
		selectedParser, confidence, detectedMeta = s.registry.Detect(filename, workingBytes)
		if selectedParser == nil || confidence < 0.2 {
			if isPDF && extractor.IsPDFEncrypted(bytes.NewReader(fileBytes)) {
				res.RequiresPassword = true
				res.Error = "Password required to open encrypted statement PDF"
				return res, nil
			}
			res.Error = fmt.Sprintf("could not auto-detect format for file '%s'. Please select a bank format.", filename)
			return res, nil
		}
	}

	res.ParserID = selectedParser.ID()
	res.ParserName = selectedParser.Name()
	res.Confidence = confidence
	res.BankName = detectedMeta.BankName
	res.AccountType = string(detectedMeta.AccountType)

	// 3. Run Parser
	parsedTxList, meta, err := selectedParser.Parse(bytes.NewReader(workingBytes), parser.ParseOptions{
		AccountID: manualAccountID,
		Password:  password,
		Filename:  filename,
	})
	if err != nil {
		if errors.Is(err, extractor.ErrPDFPasswordRequired) {
			res.RequiresPassword = true
			res.Error = err.Error()
			return res, nil
		}
		res.Error = fmt.Sprintf("Parser failed: %s", err)
		return res, nil
	}

	if len(parsedTxList) == 0 {
		res.Error = "No transactions found in statement. Please verify the file format."
		return res, nil
	}

	// 4. Fill metadata
	if meta.BankName != "" {
		res.BankName = meta.BankName
	}
	if meta.AccountType != "" {
		res.AccountType = string(meta.AccountType)
	}
	res.AccountNumber = meta.AccountNumber
	res.AccountNumberMask = meta.AccountNumberMask
	res.StartDate = meta.StartDate
	res.EndDate = meta.EndDate
	res.TotalTransactions = len(parsedTxList)
	res.OpeningBalance = meta.OpeningBalance
	res.ClosingBalance = meta.ClosingBalance
	res.TotalDebits = meta.TotalDebits
	res.TotalCredits = meta.TotalCredits
	res.TotalDueAmount = meta.TotalDueAmount
	res.MinimumDueAmount = meta.MinimumDueAmount
	res.PaymentDueDate = meta.PaymentDueDate
	res.CreditLimit = meta.CreditLimit
	res.AvailableCreditLimit = meta.AvailableCreditLimit
	res.RewardPointsBalance = meta.RewardPointsBalance
	res.CashbackEarned = meta.CashbackEarned
	res.CardNetwork = meta.CardNetwork
	res.CardVariant = meta.CardVariant
	res.AccountHolderName = meta.AccountHolderName

	// Load categories & rules for preview categorization
	rules, _ := s.db.ListRules()
	categories, _ := s.db.ListCategories()
	catMap := make(map[string]models.Category)
	for _, c := range categories {
		catMap[c.ID] = c
	}

	// Resolve Account ID for checking existing fingerprints
	var accountID string
	if manualAccountID != "" {
		accountID = manualAccountID
	} else {
		if accounts, err := s.db.ListAccounts(); err == nil {
			for _, a := range accounts {
				if a.BankName != res.BankName {
					continue
				}
				storedNumber := ""
				if a.AccountNumber != nil {
					storedNumber = *a.AccountNumber
				}
				// Match the same identity policy as GetOrCreateAccount. An exact
				// full-number match takes priority over a mask-only candidate.
				if meta.AccountNumber != "" && storedNumber == meta.AccountNumber {
					accountID = a.ID
					break
				}
				if string(a.AccountType) != res.AccountType || meta.AccountNumber != "" && storedNumber != "" {
					continue
				}
				if meta.AccountNumberMask != "" && a.AccountNumberMask == meta.AccountNumberMask ||
					meta.AccountNumber == "" && meta.AccountNumberMask == "" && storedNumber == "" && a.AccountNumberMask == "" {
					if accountID == "" {
						accountID = a.ID
					}
				}
			}
		}
	}

	// 5. Convert & Categorize Preview Transactions
	var previewTxs []models.PreviewTransactionItem
	for _, pt := range parsedTxList {
		catID := s.matchCategory(pt, rules)
		var catName, catColor, catIcon string
		if catID != nil {
			if c, ok := catMap[*catID]; ok {
				catName = c.Name
				catColor = c.ColorHex
				catIcon = c.Icon
			}
		}

		isDuplicate := false
		if accountID != "" {
			txHash := s.calculateTxHash(accountID, pt.Date, pt.Amount, pt.RawNarration, pt.ReferenceNumber, pt.TxType)
			if exists, err := s.db.CheckTxHashExists(txHash); err == nil && exists {
				isDuplicate = true
			}
		}

		if isDuplicate {
			res.ExistingTxsCount++
		} else {
			res.NewTxsCount++
		}

		previewTxs = append(previewTxs, models.PreviewTransactionItem{
			Date:            pt.Date,
			ValueDate:       pt.ValueDate,
			RawNarration:    pt.RawNarration,
			CleanedPayee:    pt.CleanedPayee,
			PaymentMode:     string(pt.PaymentMode),
			ReferenceNumber: pt.ReferenceNumber,
			TxType:          string(pt.TxType),
			Amount:          pt.Amount,
			RunningBalance:  pt.RunningBalance,
			CategoryID:      catID,
			CategoryName:    catName,
			CategoryColor:   catColor,
			CategoryIcon:    catIcon,
			UPIVPA:          pt.UPIVPA,
			IsTransfer:      pt.IsTransfer,
			IsDuplicate:     isDuplicate,
		})
	}

	res.Transactions = previewTxs
	return res, nil
}

func (s *TransactionService) calculateTxHash(accountID, date string, amount float64, narration, refNo string, txType models.TxType) string {
	raw := fmt.Sprintf("%s|%s|%.2f|%s|%s|%s", accountID, date, amount, strings.TrimSpace(narration), strings.TrimSpace(refNo), txType)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func (s *TransactionService) matchCategory(pt parser.ParsedTransaction, rules []models.CategorizationRule) *string {
	if pt.IsTransfer {
		transferCat := models.CategoryTransfersID
		return &transferCat
	}

	tx := models.Transaction{RawNarration: pt.RawNarration, CleanedPayee: pt.CleanedPayee, ReferenceNumber: pt.ReferenceNumber, UPIVPA: pt.UPIVPA, TxType: pt.TxType}
	for _, r := range rules {
		if r.MatchesTransaction(tx) {
			catID := r.TargetCategoryID
			return &catID
		}
	}

	defaultCat := "cat_others"
	return &defaultCat
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
