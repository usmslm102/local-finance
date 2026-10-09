package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"local-finance/internal/db"
	"local-finance/internal/mcp"
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"local-finance/internal/service"
	"local-finance/internal/updater"
)

type Handler struct {
	mcpManager     *mcp.Manager
	db             *db.DB
	service        *service.TransactionService
	sessionManager *SessionManager
	updaterService *updater.Service
}

func NewHandler(database *db.DB, svc *service.TransactionService) *Handler {
	manager := mcp.NewManager(database)
	return newHandler(database, svc, manager)
}

func newHandler(database *db.DB, svc *service.TransactionService, manager *mcp.Manager) *Handler {
	return &Handler{
		db:             database,
		mcpManager:     manager,
		service:        svc,
		sessionManager: NewSessionManager(),
		updaterService: updater.NewService(),
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": updater.CurrentVersion,
		"app":     "local-finance",
	})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	accounts, err := h.db.ListAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if accounts == nil {
		accounts = []models.Account{}
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *Handler) ListTransactions(c *gin.Context) {
	amountBounds := make(map[string]*float64, 2)
	for _, key := range []string{"min_amount", "max_amount"} {
		if raw := c.Query(key); raw != "" {
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				c.JSON(http.StatusBadRequest, gin.H{"error": key + " must be a finite number"})
				return
			}
			amountBounds[key] = &value
		}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 10000 {
		pageSize = 50
	}

	var isTransferPtr *bool
	if isTrfStr := c.Query("is_transfer"); isTrfStr != "" {
		val := isTrfStr == "true" || isTrfStr == "1"
		isTransferPtr = &val
	}

	filter := db.TransactionFilter{
		AccountID:  c.Query("account_id"),
		CategoryID: c.Query("category_id"),
		TxType:     c.Query("tx_type"),
		Search:     c.Query("search"),
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
		IsTransfer: isTransferPtr,
		MinAmount:  amountBounds["min_amount"],
		MaxAmount:  amountBounds["max_amount"],
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
	}

	transactions, total, err := h.db.ListTransactions(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if transactions == nil {
		transactions = []models.Transaction{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       transactions,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

func (h *Handler) GetTransaction(c *gin.Context) {
	id := c.Param("id")
	tx, err := h.db.GetTransaction(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tx)
}

func (h *Handler) UpdateTransaction(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedTx, err := h.db.UpdateTransaction(id, req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedTx)
}

func (h *Handler) UploadStatement(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing statement file in 'file' field"})
		return
	}
	defer file.Close()

	manualAccountID := c.PostForm("account_id")
	manualParserID := c.PostForm("parser_id")
	password := c.PostForm("password")

	result, err := h.service.ImportStatement(header.Filename, file, manualAccountID, manualParserID, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) PreviewStatement(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing statement file in 'file' field"})
		return
	}
	defer file.Close()

	manualAccountID := c.PostForm("account_id")
	manualParserID := c.PostForm("parser_id")
	password := c.PostForm("password")

	result, err := h.service.PreviewStatement(header.Filename, file, manualAccountID, manualParserID, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) PreviewBatchStatements(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded in 'files' field"})
		return
	}

	manualAccountID := c.PostForm("account_id")
	manualParserID := c.PostForm("parser_id")
	defaultPassword := c.PostForm("password")

	passwordsMap := make(map[string]string)
	if pwJson := c.PostForm("passwords"); pwJson != "" {
		_ = json.Unmarshal([]byte(pwJson), &passwordsMap)
	}

	var results []*models.StatementPreviewResult
	for _, fileHeader := range files {
		f, err := fileHeader.Open()
		if err != nil {
			results = append(results, &models.StatementPreviewResult{
				Filename: fileHeader.Filename,
				Error:    fmt.Sprintf("Failed to open file: %v", err),
			})
			continue
		}

		pwd := defaultPassword
		if p, exists := passwordsMap[fileHeader.Filename]; exists && p != "" {
			pwd = p
		}

		res, err := h.service.PreviewStatement(fileHeader.Filename, f, manualAccountID, manualParserID, pwd)
		f.Close()
		if err != nil {
			results = append(results, &models.StatementPreviewResult{
				Filename: fileHeader.Filename,
				Error:    err.Error(),
			})
		} else {
			results = append(results, res)
		}
	}

	c.JSON(http.StatusOK, results)
}

func (h *Handler) UploadBatchStatements(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
		return
	}

	manualAccountID := c.PostForm("account_id")
	manualParserID := c.PostForm("parser_id")
	defaultPassword := c.PostForm("password")

	passwordsMap := make(map[string]string)
	if pwJson := c.PostForm("passwords"); pwJson != "" {
		_ = json.Unmarshal([]byte(pwJson), &passwordsMap)
	}

	var results []*models.ImportResult
	var errors []string

	for _, fileHeader := range files {
		f, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to open (%v)", fileHeader.Filename, err))
			continue
		}

		pwd := defaultPassword
		if p, exists := passwordsMap[fileHeader.Filename]; exists && p != "" {
			pwd = p
		}

		res, err := h.service.ImportStatement(fileHeader.Filename, f, manualAccountID, manualParserID, pwd)
		f.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", fileHeader.Filename, err))
		} else {
			results = append(results, res)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"errors":  errors,
		"success": len(results),
		"failed":  len(errors),
	})
}

func (h *Handler) ListStatementImports(c *gin.Context) {
	imports, err := h.db.ListStatementImports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if imports == nil {
		imports = []models.StatementImport{}
	}
	c.JSON(http.StatusOK, imports)
}

func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.db.ListCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if categories == nil {
		categories = []models.Category{}
	}
	c.JSON(http.StatusOK, categories)
}

func (h *Handler) ListRules(c *gin.Context) {
	rules, err := h.db.ListRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rules == nil {
		rules = []models.CategorizationRule{}
	}
	c.JSON(http.StatusOK, rules)
}

func (h *Handler) GetAnalyticsOverview(c *gin.Context) {
	overview, err := h.db.GetAnalyticsOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *Handler) GetCashFlowIntelligence(c *gin.Context) {
	period := c.Query("period")
	resp, err := h.db.GetCashFlowIntelligence(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetSalaryInsights(c *gin.Context) {
	resp, err := h.db.GetSalaryInsights()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetWrappedStory(c *gin.Context) {
	year := c.Query("year")
	story, err := h.db.GetWrappedStory(year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, story)
}

func (h *Handler) ListParsers(c *gin.Context) {
	parsers := parser.DefaultRegistry.List()
	type ParserInfo struct {
		ID             string                 `json:"id"`
		Name           string                 `json:"name"`
		SupportedTypes []parser.StatementType `json:"supported_types"`
	}
	var res []ParserInfo
	for _, p := range parsers {
		res = append(res, ParserInfo{
			ID:             p.ID(),
			Name:           p.Name(),
			SupportedTypes: p.SupportedTypes(),
		})
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ResetDatabase(c *gin.Context) {
	if err := h.mcpManager.Revoke(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to revoke MCP access"})
		return
	}
	if err := h.db.ResetDatabase(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "database reset successfully: cleared transactions, accounts, and imports",
	})
}

func (h *Handler) ListCreditCardBills(c *gin.Context) {
	accountID := c.Query("account_id")
	bills, err := h.db.ListCreditCardBills(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bills == nil {
		bills = []models.CreditCardBill{}
	}
	c.JSON(http.StatusOK, bills)
}

func (h *Handler) GetDatabaseInfo(c *gin.Context) {
	info, err := h.db.GetDatabaseInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *Handler) BackupDatabase(c *gin.Context) {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("local_finance_backup_%s.db", time.Now().Format("20060102_150405")))
	defer os.Remove(tempFile)

	if err := h.db.BackupTo(tempFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(tempFile)))
	c.Header("Content-Type", "application/x-sqlite3")
	c.File(tempFile)
}

func (h *Handler) RestoreDatabase(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing database backup file in upload"})
		return
	}
	defer file.Close()

	if err := h.mcpManager.Revoke(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to revoke MCP access"})
		return
	}
	if err := h.db.RestoreFrom(file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.mcpManager.Start() // Reload restored preferences; restored MCP access is always disabled.
	info, _ := h.db.GetDatabaseInfo()
	c.JSON(http.StatusOK, gin.H{
		"message": "Database successfully restored from backup",
		"info":    info,
	})
}

func (h *Handler) ExportDatabaseJSON(c *gin.Context) {
	data, err := h.db.ExportAllDataJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("local_finance_export_%s.json", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, data)
}

func (h *Handler) ExportTransactionsCSV(c *gin.Context) {
	filename := fmt.Sprintf("local_finance_transactions_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv")

	if err := h.db.ExportTransactionsCSV(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func (h *Handler) GetSubscriptionsSummary(c *gin.Context) {
	summary, err := h.db.GetSubscriptionsSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) ScanSubscriptions(c *gin.Context) {
	subService := service.NewSubscriptionService(h.db)
	summary, err := subService.ScanAndDetectSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	var req models.UpsertSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	freq := req.Frequency
	if freq == "" {
		freq = models.FrequencyMonthly
	}
	status := req.Status
	if status == "" {
		status = models.SubscriptionStatusActive
	}

	sub := &models.Subscription{
		Name:            req.Name,
		MerchantPattern: req.MerchantPattern,
		CategoryID:      req.CategoryID,
		AccountID:       req.AccountID,
		Frequency:       freq,
		ExpectedAmount:  req.ExpectedAmount,
		Currency:        "INR",
		BillingDay:      req.BillingDay,
		NextDueDate:     req.NextDueDate,
		Status:          status,
		IsAutoDetected:  false,
		Notes:           req.Notes,
	}

	if err := h.db.CreateSubscription(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *Handler) UpdateSubscription(c *gin.Context) {
	id := c.Param("id")
	var req models.UpsertSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.db.GetSubscription(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	existing.Name = req.Name
	existing.MerchantPattern = req.MerchantPattern
	existing.CategoryID = req.CategoryID
	existing.AccountID = req.AccountID
	if req.Frequency != "" {
		existing.Frequency = req.Frequency
	}
	existing.ExpectedAmount = req.ExpectedAmount
	existing.BillingDay = req.BillingDay
	existing.NextDueDate = req.NextDueDate
	if req.Status != "" {
		existing.Status = req.Status
	}
	existing.Notes = req.Notes

	if err := h.db.UpdateSubscription(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

func (h *Handler) DeleteSubscription(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription deleted successfully"})
}

func (h *Handler) CreateRule(c *gin.Context) {
	var rule models.CategorizationRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	if rule.MatchPattern == "" || rule.TargetCategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MatchPattern and TargetCategoryID are required"})
		return
	}
	if rule.MatchField == "" {
		rule.MatchField = "cleaned_payee"
	}
	if rule.MatchType == "" {
		rule.MatchType = "CONTAINS"
	}
	rule.IsActive = true

	if err := h.db.CreateRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *Handler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.CategorizationRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	rule.ID = id

	if err := h.db.UpdateRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "rule deleted successfully"})
}

func (h *Handler) ReapplyRules(c *gin.Context) {
	count, err := h.db.ReapplyRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated_count": count, "message": fmt.Sprintf("Successfully re-categorized %d transactions", count)})
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var cat models.Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	if cat.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category name is required"})
		return
	}
	if err := h.db.CreateCategory(&cat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteCategory(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}

// -------------------------------------------------------------
// Credit Card Reward & Best-Card Recommendation Endpoints
// -------------------------------------------------------------

func (h *Handler) GetCardPortfolioOverview(c *gin.Context) {
	overview, err := h.db.GetCardPortfolioOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *Handler) RecommendBestCard(c *gin.Context) {
	var req models.CardRecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recommendation payload: " + err.Error()})
		return
	}

	recService := service.NewCardRecommendationService(h.db)
	recommendations, err := recService.RecommendBestCards(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"merchant":        req.Merchant,
		"category":        req.Category,
		"spend_amount":    req.Amount,
		"recommendations": recommendations,
	})
}

func (h *Handler) UpdateCardMetadata(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateCardMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	if err := h.db.UpdateCardMetadata(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "card updated successfully"})
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	if err := h.db.UpdateAccount(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "account updated successfully"})
}

func (h *Handler) ListCardRewardRules(c *gin.Context) {
	accountID := c.Query("account_id")
	rules, err := h.db.ListCardRewardRules(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *Handler) CreateCardRewardRule(c *gin.Context) {
	var rule models.CardRewardRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	if rule.AccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
		return
	}
	if rule.MerchantPattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "merchant_pattern is required"})
		return
	}

	if err := h.db.CreateCardRewardRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *Handler) UpdateCardRewardRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.CardRewardRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	rule.ID = id
	if err := h.db.UpdateCardRewardRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *Handler) DeleteCardRewardRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteCardRewardRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reward rule deleted successfully"})
}

// ==========================================
// CATEGORY BUDGET HANDLERS
// ==========================================

func (h *Handler) GetBudgetSummary(c *gin.Context) {
	month := c.Query("month")
	summary, err := h.db.GetCategoryBudgetSummary(month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) UpsertCategoryBudget(c *gin.Context) {
	var req models.UpsertCategoryBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	if err := h.db.UpsertCategoryBudget(req.CategoryID, req.Month, req.MonthlyLimit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	summary, err := h.db.GetCategoryBudgetSummary(req.Month)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "budget saved successfully"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) DeleteCategoryBudget(c *gin.Context) {
	categoryID := c.Param("categoryId")
	month := c.Query("month")

	if err := h.db.DeleteCategoryBudget(categoryID, month); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "budget removed successfully"})
}

// ==========================================
// TRANSFER RECONCILIATION HANDLERS
// ==========================================

func (h *Handler) GetReconciliationSummary(c *gin.Context) {
	reconService := service.NewReconciliationService(h.db)
	summary, err := reconService.GetSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) ScanReconciliation(c *gin.Context) {
	reconService := service.NewReconciliationService(h.db)
	autoLinked, walletExcluded, err := reconService.ScanAndAutoReconcile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":               fmt.Sprintf("Successfully scanned ledger: %d transfer pairs linked, %d wallet loads excluded", autoLinked, walletExcluded),
		"auto_linked_count":     autoLinked,
		"wallet_excluded_count": walletExcluded,
	})
}

func (h *Handler) LinkTransferPair(c *gin.Context) {
	var req models.LinkTransferPairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	reconService := service.NewReconciliationService(h.db)
	if err := reconService.LinkPair(req.DebitTxID, req.CreditTxID, req.MatchReason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "transfer pair linked successfully"})
}

func (h *Handler) UnlinkTransferPair(c *gin.Context) {
	var req models.UnlinkTransferPairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	reconService := service.NewReconciliationService(h.db)
	if err := reconService.UnlinkPair(req.TxID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "transfer pair unlinked successfully"})
}

func (h *Handler) ToggleExcludeTransaction(c *gin.Context) {
	id := c.Param("id")
	newStatus, err := h.db.ToggleExcludeTransaction(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":     "transaction exclusion toggled",
		"is_excluded": newStatus,
	})
}

// ==========================================
// MERCHANT INTELLIGENCE HANDLERS
// ==========================================

func (h *Handler) ListMerchants(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	sortBy := c.Query("sort_by")

	res, err := h.db.ListMerchants(search, category, sortBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetMerchantProfile(c *gin.Context) {
	name := c.Param("name")
	profile, err := h.db.GetMerchantProfile(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

// ==========================================
// SAMPLE STATEMENT FIXTURE HANDLERS
// ==========================================

type SampleStatementInfo struct {
	Category string `json:"category"`
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Bank     string `json:"bank"`
	Type     string `json:"type"`
	Format   string `json:"format"`
	Path     string `json:"path"`
}

func (h *Handler) ListSampleStatements(c *gin.Context) {
	samples := []SampleStatementInfo{
		{Category: "Credit Cards", Filename: "HDFC_Regalia_Credit_Card.pdf", Title: "HDFC Regalia Credit Card (PDF)", Bank: "HDFC Bank", Type: "CREDIT_CARD", Format: "PDF", Path: "samples/credit_cards/HDFC_Regalia_Credit_Card.pdf"},
		{Category: "Credit Cards", Filename: "ICICI_Amazon_Pay_Credit_Card.pdf", Title: "ICICI Amazon Pay Credit Card (PDF)", Bank: "ICICI Bank", Type: "CREDIT_CARD", Format: "PDF", Path: "samples/credit_cards/ICICI_Amazon_Pay_Credit_Card.pdf"},
		{Category: "Credit Cards", Filename: "Axis_Flipkart_Credit_Card.pdf", Title: "Axis Flipkart Credit Card (PDF)", Bank: "Axis Bank", Type: "CREDIT_CARD", Format: "PDF", Path: "samples/credit_cards/Axis_Flipkart_Credit_Card.pdf"},
		{Category: "Credit Cards", Filename: "HDFC_Swiggy_Credit_Card.pdf", Title: "HDFC Swiggy Credit Card (PDF)", Bank: "HDFC Bank", Type: "CREDIT_CARD", Format: "PDF", Path: "samples/credit_cards/HDFC_Swiggy_Credit_Card.pdf"},
		{Category: "Credit Cards", Filename: "HDFC_RuPay_Credit_Card.pdf", Title: "HDFC RuPay Credit Card (PDF)", Bank: "HDFC Bank", Type: "CREDIT_CARD", Format: "PDF", Path: "samples/credit_cards/HDFC_RuPay_Credit_Card.pdf"},
		{Category: "Credit Cards", Filename: "HDFC_Credit_Card_Statement.csv", Title: "HDFC Credit Card (CSV)", Bank: "HDFC Bank", Type: "CREDIT_CARD", Format: "CSV", Path: "samples/credit_cards/HDFC_Credit_Card_Statement.csv"},
		{Category: "Savings Account", Filename: "HDFC_Savings_Account_Statement.pdf", Title: "HDFC Savings Account (PDF)", Bank: "HDFC Bank", Type: "SAVINGS", Format: "PDF", Path: "samples/savings/HDFC_Savings_Account_Statement.pdf"},
		{Category: "Current Account", Filename: "HDFC_Current_Account_Statement.pdf", Title: "HDFC Current Account (PDF)", Bank: "HDFC Bank", Type: "CURRENT", Format: "PDF", Path: "samples/savings/HDFC_Current_Account_Statement.pdf"},
		{Category: "Savings Account", Filename: "HDFC_Savings_Account_Statement.csv", Title: "HDFC Savings Account (CSV)", Bank: "HDFC Bank", Type: "SAVINGS", Format: "CSV", Path: "samples/savings/HDFC_Savings_Account_Statement.csv"},
	}
	c.JSON(http.StatusOK, samples)
}

func (h *Handler) ServeSampleStatement(c *gin.Context) {
	relPath := c.Query("path")
	if relPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter required"})
		return
	}
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") || !strings.HasPrefix(cleanPath, "samples/") {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid sample file path"})
		return
	}
	c.File(cleanPath)
}
