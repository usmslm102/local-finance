package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"local-finance/internal/api"
	"local-finance/internal/db"
	"local-finance/internal/models"
	"local-finance/internal/service"
)

func setupTestRouter(t *testing.T) (*api.Handler, http.Handler, *db.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "api_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test_api.db")

	database, err := db.NewDB(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open DB: %v", err)
	}

	svc := service.NewTransactionService(database)
	router := api.SetupRouter(database, svc, nil)

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return api.NewHandler(database, svc), router, database, cleanup
}

func TestAPIRoutes(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()

	// Seed test account
	acc, err := database.GetOrCreateAccount("HDFC Bank", models.AccountTypeSavings, "", "1234", "CUST001", "HDFC0001", "Main Branch", "", "", "RAHUL SHARMA", nil)
	if err != nil {
		t.Fatalf("Failed to seed account: %v", err)
	}

	// Seed transactions
	t1 := &models.Transaction{
		AccountID:       acc.ID,
		TxHash:          "hash_api_001",
		TxDate:          "2026-08-01",
		RawNarration:    "UPI-SWIGGY-swiggy@okhdfcbank-423589214781-PAY",
		CleanedPayee:    "Swiggy",
		PaymentMode:     models.PaymentModeUPI,
		ReferenceNumber: "423589214781",
		TxType:          models.TxTypeDebit,
		Amount:          850.00,
	}
	_, err = database.UpsertTransaction(t1)
	if err != nil {
		t.Fatalf("Failed to seed t1: %v", err)
	}

	t2 := &models.Transaction{
		AccountID:       acc.ID,
		TxHash:          "hash_api_002",
		TxDate:          "2026-08-01",
		RawNarration:    "SALARY CREDIT TECH CORP",
		CleanedPayee:    "Salary Credit",
		PaymentMode:     models.PaymentModeSalary,
		ReferenceNumber: "SAL999888",
		TxType:          models.TxTypeCredit,
		Amount:          95000.00,
	}
	_, err = database.UpsertTransaction(t2)
	if err != nil {
		t.Fatalf("Failed to seed t2: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		url        string
		body       interface{}
		expectCode int
	}{
		{
			name:       "Health Check",
			method:     "GET",
			url:        "/api/health",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Accounts",
			method:     "GET",
			url:        "/api/accounts",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Transactions",
			method:     "GET",
			url:        "/api/transactions",
			expectCode: http.StatusOK,
		},
		{
			name:       "Filter Transactions",
			method:     "GET",
			url:        "/api/transactions?tx_type=DEBIT&search=Swiggy",
			expectCode: http.StatusOK,
		},
		{
			name:       "Analytics Overview",
			method:     "GET",
			url:        "/api/analytics/overview",
			expectCode: http.StatusOK,
		},
		{
			name:       "Analytics Cash Flow Intelligence",
			method:     "GET",
			url:        "/api/analytics/cashflow?period=2026-08",
			expectCode: http.StatusOK,
		},
		{
			name:       "Analytics Wrapped",
			method:     "GET",
			url:        "/api/analytics/wrapped?year=2026",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Categories",
			method:     "GET",
			url:        "/api/categories",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Rules",
			method:     "GET",
			url:        "/api/rules",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Parsers",
			method:     "GET",
			url:        "/api/parsers",
			expectCode: http.StatusOK,
		},
		{
			name:       "Credit Card Bills",
			method:     "GET",
			url:        "/api/credit-cards/bills",
			expectCode: http.StatusOK,
		},
		{
			name:       "Card Portfolio",
			method:     "GET",
			url:        "/api/cards/portfolio",
			expectCode: http.StatusOK,
		},
		{
			name:       "Recommend Best Card",
			method:     "POST",
			url:        "/api/cards/recommend",
			body:       map[string]interface{}{"merchant": "SWIGGY", "amount": 1000},
			expectCode: http.StatusOK,
		},
		{
			name:       "Subscriptions Summary",
			method:     "GET",
			url:        "/api/subscriptions",
			expectCode: http.StatusOK,
		},
		{
			name:       "Budgets Summary",
			method:     "GET",
			url:        "/api/budgets?month=2026-08",
			expectCode: http.StatusOK,
		},
		{
			name:       "Reconciliation Summary",
			method:     "GET",
			url:        "/api/reconciliation/summary",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Merchants",
			method:     "GET",
			url:        "/api/merchants",
			expectCode: http.StatusOK,
		},
		{
			name:       "Get Merchant Profile",
			method:     "GET",
			url:        "/api/merchants/Swiggy",
			expectCode: http.StatusOK,
		},
		{
			name:       "Database Info",
			method:     "GET",
			url:        "/api/database/info",
			expectCode: http.StatusOK,
		},
		{
			name:       "Export Transactions CSV",
			method:     "GET",
			url:        "/api/database/export/csv",
			expectCode: http.StatusOK,
		},
		{
			name:       "Export Database JSON",
			method:     "GET",
			url:        "/api/database/export/json",
			expectCode: http.StatusOK,
		},
		{
			name:       "List Samples",
			method:     "GET",
			url:        "/api/samples",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBytes, _ := json.Marshal(tt.body)
				req = localTestRequest(tt.method, tt.url, bytes.NewBuffer(jsonBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = localTestRequest(tt.method, tt.url, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("%s %s expected status %d, got %d. Body: %s", tt.method, tt.url, tt.expectCode, w.Code, w.Body.String())
			}
		})
	}
}
