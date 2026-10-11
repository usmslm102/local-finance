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
