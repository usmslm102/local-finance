package db

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"local-finance/internal/models"
)

func TestAnalyticsOverviewReturnsLatestRecordedMonthsChronologically(t *testing.T) {
	for _, tc := range []struct {
		name            string
		count, interval int
	}{
		{name: "empty"},
		{name: "short history", count: 5, interval: 1},
		{name: "exactly twelve months", count: 12, interval: 1},
		{name: "long history", count: 26, interval: 1},
		{name: "gaps in long history", count: 26, interval: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewDB(filepath.Join(t.TempDir(), "overview.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			account, err := d.GetOrCreateAccount("Test Bank", models.AccountTypeSavings, "", "XX1234", "", "", "", "", "", "", nil)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
			for i := 0; i < tc.count; i++ {
				date := start.AddDate(0, i*tc.interval, 0).Format(time.DateOnly)
				for _, tx := range []*models.Transaction{
					{TxHash: fmt.Sprintf("income-%d", i), TxType: models.TxTypeCredit, Amount: float64(1000 + i)},
					{TxHash: fmt.Sprintf("spending-%d", i), TxType: models.TxTypeDebit, Amount: float64(100 + i)},
				} {
					tx.AccountID, tx.TxDate = account.ID, date
					if _, err := d.UpsertTransaction(tx); err != nil {
						t.Fatal(err)
					}
				}
			}
			// A newer month consisting only of non-financial activity must not
			// displace a month that contains income or spending.
			transferCategory := models.CategoryTransfersID
			for _, tx := range []*models.Transaction{
				{TxHash: "transfer", TxType: models.TxTypeCredit, Amount: 999999, IsTransfer: true},
				{TxHash: "excluded", TxType: models.TxTypeDebit, Amount: 999999, IsExcluded: true},
				{TxHash: "transfer-category", TxType: models.TxTypeCredit, Amount: 999999, CategoryID: &transferCategory},
			} {
				tx.AccountID = account.ID
				tx.TxDate = start.AddDate(0, tc.count*max(1, tc.interval)+1, 0).Format(time.DateOnly)
				if _, err := d.UpsertTransaction(tx); err != nil {
					t.Fatal(err)
				}
			}
			overview, err := d.GetAnalyticsOverview()
			if err != nil {
				t.Fatal(err)
			}
			wantCount := min(tc.count, 12)
			if len(overview.MonthlyTrends) != wantCount {
				t.Fatalf("got %d months, want %d", len(overview.MonthlyTrends), wantCount)
			}
			for index, trend := range overview.MonthlyTrends {
				i := tc.count - wantCount + index
				wantMonth := start.AddDate(0, i*tc.interval, 0).Format("2006-01")
				if trend.Month != wantMonth || trend.Income != float64(1000+i) || trend.Expense != float64(100+i) || trend.Net != 900 {
					t.Errorf("row %d: got %+v; want month %s, income %d, expense %d, net 900", index, trend, wantMonth, 1000+i, 100+i)
				}
			}
		})
	}
}

func TestAnalyticsOverviewRangeFiltersFlowsInclusively(t *testing.T) {
	d, err := NewDB(filepath.Join(t.TempDir(), "range.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	account, err := d.GetOrCreateAccount("Test Bank", models.AccountTypeSavings, "", "XX1234", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, date := range []string{"2024-01-31", "2024-02-01", "2024-02-29", "2024-03-01"} {
		for _, kind := range []models.TxType{models.TxTypeCredit, models.TxTypeDebit} {
			tx := &models.Transaction{AccountID: account.ID, TxDate: date, TxHash: fmt.Sprintf("%d-%s", i, kind), TxType: kind, Amount: 100, CleanedPayee: "Fictional Shop"}
			if _, err := d.UpsertTransaction(tx); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, err := d.GetAnalyticsOverviewRange("2024-02-01", "2024-02-29")
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalIncome != 200 || got.TotalExpense != 200 || got.TotalTransactions != 4 {
		t.Fatalf("incorrect range totals: %+v", got)
	}
	if len(got.MonthlyTrends) != 1 || got.MonthlyTrends[0].Month != "2024-02" {
		t.Fatalf("incorrect trends: %+v", got.MonthlyTrends)
	}
	if len(got.CategoryBreakdown) != 1 || got.CategoryBreakdown[0].TotalAmount != 200 || got.CategoryBreakdown[0].TxCount != 2 || got.CategoryBreakdown[0].Percentage != 100 {
		t.Fatalf("incorrect categories: %+v", got.CategoryBreakdown)
	}
	if len(got.TopPayees) != 1 || got.TopPayees[0].TotalSpent != 200 || got.TopPayees[0].TxCount != 2 {
		t.Fatalf("incorrect payees: %+v", got.TopPayees)
	}
	all, err := d.GetAnalyticsOverview()
	if err != nil {
		t.Fatal(err)
	}
	if all.TotalExpense != 400 || all.TotalTransactions != 8 || all.TotalAccounts != got.TotalAccounts || all.TotalBankLiquidity != got.TotalBankLiquidity {
		t.Fatalf("incorrect all-time or current snapshot: %+v", all)
	}
	empty, err := d.GetAnalyticsOverviewRange("2025-01-01", "2025-01-31")
	if err != nil {
		t.Fatal(err)
	}
	if empty.TotalIncome != 0 || empty.TotalExpense != 0 || empty.TotalTransactions != 0 || len(empty.MonthlyTrends) != 0 {
		t.Fatalf("incorrect empty range: %+v", empty)
	}
	for _, dates := range [][2]string{{"2024-02-30", ""}, {"", "invalid"}, {"2024-03-01", "2024-02-01"}} {
		if _, err := d.GetAnalyticsOverviewRange(dates[0], dates[1]); err == nil {
			t.Fatalf("accepted invalid range %v", dates)
		}
	}
}
