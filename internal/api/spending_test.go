package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"net/url"
	"testing"

	"local-finance/internal/models"
)

func TestSpendingDrilldownMatchesOverview(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	account, err := database.GetOrCreateAccount("Fictional Bank", models.AccountTypeSavings, "", "XX1234", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(name, date, category string, amount float64, kind models.TxType, excluded, transfer bool) string {
		t.Helper()
		transaction := &models.Transaction{AccountID: account.ID, TxHash: name, TxDate: date,
			RawNarration: name, CleanedPayee: name, Amount: amount, TxType: kind, IsExcluded: excluded, IsTransfer: transfer}
		if category != "" {
			transaction.CategoryID = &category
		}
		if _, err := database.UpsertTransaction(transaction); err != nil {
			t.Fatal(err)
		}
		return transaction.ID
	}
	bankID := add("Shared dinner", "2026-08-28", "cat_food", 200, models.TxTypeDebit, false, false)
	add("Lunch", "2026-08-29", "cat_food", 30, models.TxTypeDebit, false, false)
	add("Uncategorized purchase", "2026-08-10", "", 12, models.TxTypeDebit, false, false)
	add("Excluded", "2026-08-31", "cat_food", 1000, models.TxTypeDebit, true, false)
	add("Refund", "2026-08-31", "cat_food", 5000, models.TxTypeCredit, false, false)
	add("Transfer", "2026-08-31", "cat_food", 700, models.TxTypeDebit, false, true)
	add("Transfer category", "2026-08-31", models.CategoryTransfersID, 100, models.TxTypeDebit, false, false)
	add("Zero", "2026-08-31", "cat_food", 0, models.TxTypeDebit, false, false)
	add("Negative", "2026-08-31", "cat_food", -5, models.TxTypeDebit, false, false)
	add("Previous month", "2026-07-31", "cat_food", 50, models.TxTypeDebit, false, false)
	add("Next month", "2026-09-01", "cat_food", 70, models.TxTypeDebit, false, false)
	for _, entry := range []struct {
		id, transaction string
		share           int
	}{
		{"linked-share", bankID, 8000}, {"standalone-share", "", 2500},
	} {
		if _, err := database.Exec(`INSERT INTO splitwise_entries
			(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,transaction_id,category_id)
			VALUES(?, 'Fictional group', 'Fictional person', '2026-08-28', ?, 'Food & Dining', 20000, 0, ?, 'EXPENSE', 'CONFIRMED', NULLIF(?, ''), 'cat_food')`,
			entry.id, entry.id, entry.share, entry.transaction); err != nil {
			t.Fatal(err)
		}
	}
	for _, period := range []struct {
		name, start, end string
		expected         float64
	}{
		{"calendar month", "2026-08-01", "2026-08-31", 147}, {"all time", "", "", 267},
	} {
		t.Run(period.name, func(t *testing.T) {
			overview, err := database.GetAnalyticsOverviewRange(period.start, period.end)
			if err != nil || overview.TotalExpense != period.expected {
				t.Fatalf("overview: %+v %v", overview, err)
			}
			cashflowPeriod := "ALL"
			if period.start != "" {
				cashflowPeriod = period.start[:7]
			}
			flow, err := database.GetCashFlowIntelligence(cashflowPeriod)
			if err != nil || flow.Summary.TotalOutflow != overview.TotalExpense {
				t.Fatalf("cashflow totals: %+v %v", flow, err)
			}
			for _, category := range overview.CategoryBreakdown {
				found := false
				for _, node := range flow.Sankey.Nodes {
					if node.ID == "cat_"+category.CategoryID {
						found = node.TotalValue == category.TotalAmount && node.TxCount == category.TxCount
					}
				}
				if !found {
					t.Fatalf("Sankey category differs from purchase breakdown: %+v", category)
				}

				var sum float64
				count := 0
				for page := 1; ; page++ {
					query := url.Values{"spending_only": {"true"}, "category_id": {category.CategoryID},
						"start_date": {period.start}, "end_date": {period.end}, "page_size": {"1"}, "page": {fmt.Sprint(page)}}
					response := httptest.NewRecorder()
					router.ServeHTTP(response, localTestRequest("GET", "/api/transactions?"+query.Encode(), nil))
					var result struct {
						Items      []models.Transaction `json:"items"`
						Total      int                  `json:"total"`
						TotalPages int                  `json:"total_pages"`
					}
					if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != 200 {
						t.Fatalf("drilldown: %d %s %v", response.Code, response.Body, err)
					}
					if result.Total != category.TxCount || len(result.Items) != 1 {
						t.Fatalf("pagination mismatch: %+v category=%+v", result, category)
					}
					count += len(result.Items)
					for _, item := range result.Items {
						sum += item.Amount
					}
					if page >= result.TotalPages {
						break
					}
				}
				if count != category.TxCount || math.Abs(sum-category.TotalAmount) > 0.001 {
					t.Fatalf("category evidence differs from overview: count=%d amount=%v category=%+v", count, sum, category)
				}
			}
		})
	}
	bank, err := database.GetTransaction(bankID)
	if err != nil || bank.Amount != 200 {
		t.Fatalf("gross bank amount changed: %+v %v", bank, err)
	}
	request := localTestRequest("POST", "/api/auth/setup", bytes.NewBufferString(`{"password":"fictional-password"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal(response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, localTestRequest("GET", "/api/transactions?spending_only=true", nil))
	if response.Code != 401 {
		t.Fatalf("spending accessible while locked: %d", response.Code)
	}
}
