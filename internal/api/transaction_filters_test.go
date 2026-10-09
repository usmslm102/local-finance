package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"local-finance/internal/models"
)

func TestTransactionAmountFiltersBeforePagination(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	account, err := database.GetOrCreateAccount("Test Bank", models.AccountTypeSavings, "", "1234", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	// The first unfiltered page contains only small amounts; matching rows are older.
	for i, amount := range []float64{10, 20, 100, 200, 300, 0} {
		_, err := database.UpsertTransaction(&models.Transaction{
			AccountID: account.ID, TxHash: fmt.Sprintf("amount-filter-%d", i),
			TxDate: fmt.Sprintf("2026-08-%02d", 10-i), RawNarration: "Test payment",
			CleanedPayee: "Test", TxType: models.TxTypeDebit, Amount: amount,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name, query  string
		total, pages int
		amounts      []float64
	}{
		{"minimum", "min_amount=100", 3, 2, []float64{100, 200}},
		{"maximum", "max_amount=20", 3, 2, []float64{10, 20}},
		{"inclusive range", "min_amount=100&max_amount=200", 2, 1, []float64{100, 200}},
		{"second filtered page", "min_amount=100&page=2", 3, 2, []float64{300}},
		{"zero maximum", "max_amount=0", 1, 1, []float64{0}},
		{"combined filters", "min_amount=100&tx_type=DEBIT&search=Test&start_date=2026-08-07&end_date=2026-08-08", 2, 1, []float64{100, 200}},
		{"different type", "min_amount=100&tx_type=CREDIT", 0, 0, nil},
		{"no matches", "min_amount=301", 0, 0, nil},
		{"no bounds", "", 6, 3, []float64{10, 20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/transactions?page_size=2&"+tt.query, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			var result struct {
				Items      []models.Transaction `json:"items"`
				Total      int                  `json:"total"`
				TotalPages int                  `json:"total_pages"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Total != tt.total || result.TotalPages != tt.pages || len(result.Items) != len(tt.amounts) {
				t.Fatalf("unexpected filtered response: %+v", result)
			}
			for i, amount := range tt.amounts {
				if result.Items[i].Amount != amount {
					t.Errorf("row %d: amount %v, want %v", i, result.Items[i].Amount, amount)
				}
			}
		})
	}
	for _, value := range []string{"invalid", "NaN", "Inf", "1e999"} {
		for _, key := range []string{"min_amount", "max_amount"} {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/transactions?"+key+"="+value, nil))
			if response.Code != http.StatusBadRequest {
				t.Errorf("%s=%s: status %d, want 400", key, value, response.Code)
			}
		}
	}
}
