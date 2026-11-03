package api_test

import (
	"encoding/json"
	"fmt"
	"local-finance/internal/models"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestTransactionPagesAreBoundedAndStableForTiedDates(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	account, err := database.GetOrCreateAccount("Test Bank", models.AccountTypeSavings, "", "1234", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var expected []string
	for i := 0; i < 126; i++ {
		narration := "Pagination fixture"
		if i == 125 {
			narration = "Other fixture"
		}
		tx := models.Transaction{AccountID: account.ID, TxHash: fmt.Sprintf("pagination-%d", i), TxDate: "2026-10-01", RawNarration: narration, CleanedPayee: narration, TxType: models.TxTypeDebit, Amount: float64(i + 1)}
		if _, err := database.UpsertTransaction(&tx); err != nil {
			t.Fatal(err)
		}
		if i < 125 {
			expected = append(expected, tx.ID)
		}
	}
	if _, err := database.Exec("UPDATE transactions SET created_at = '2026-10-01 12:00:00'"); err != nil {
		t.Fatal(err)
	}
	slices.Sort(expected)
	slices.Reverse(expected)
	var got []string
	for page := 1; page <= 3; page++ {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, localTestRequest(http.MethodGet, fmt.Sprintf("/api/transactions?search=Pagination&page=%d", page), nil))
		if response.Code != http.StatusOK {
			t.Fatal(response.Body.String())
		}
		var result struct {
			Items      []models.Transaction `json:"items"`
			Total      int                  `json:"total"`
			Page       int                  `json:"page"`
			PageSize   int                  `json:"page_size"`
			TotalPages int                  `json:"total_pages"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := 50
		if page == 3 {
			want = 25
		}
		if result.Total != 125 || result.TotalPages != 3 || result.PageSize != 50 || result.Page != page || len(result.Items) != want {
			t.Fatalf("invalid page metadata/size: %+v", result)
		}
		for _, tx := range result.Items {
			got = append(got, tx.ID)
		}
	}
	if !slices.Equal(got, expected) {
		t.Fatal("pagination lost, duplicated or reordered transactions with tied dates and timestamps")
	}
}
