package db

import (
	"fmt"
	"strings"
	"testing"

	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func TestSplitwiseFinalReviewMembershipStoredOnce(t *testing.T) {
	d := safetyDB(t)
	var csv strings.Builder
	csv.WriteString("Date,Description,Category,Cost,Currency,Sanjay Example")
	for i := 1; i < 100; i++ {
		fmt.Fprintf(&csv, ",Member %03d", i)
	}
	csv.WriteByte('\n')
	for row := 0; row < 20; row++ {
		fmt.Fprintf(&csv, "2026-09-01,Demo expense %d,General,100,INR,-1,1", row)
		for i := 2; i < 100; i++ {
			csv.WriteString(",0")
		}
		csv.WriteByte('\n')
	}
	entries, err := parser.ParseSplitwise(strings.NewReader(csv.String()), strings.Repeat("G", 200), "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	var groups, repeatedHeaders int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM splitwise_groups`).Scan(&groups); err != nil {
		t.Fatal(err)
	}
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM splitwise_entries WHERE member_names!='[]'`).Scan(&repeatedHeaders); err != nil {
		t.Fatal(err)
	}
	if groups != 1 || repeatedHeaders != 0 {
		t.Fatalf("membership not normalized: groups=%d repeated headers=%d", groups, repeatedHeaders)
	}
	loaded, err := d.ListSplitwise()
	if err != nil || len(loaded) != 20 {
		t.Fatalf("entries: %d %v", len(loaded), err)
	}
	for _, entry := range loaded {
		if len(entry.Members) != 100 {
			t.Fatalf("public entry lost group membership: %d", len(entry.Members))
		}
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil || len(members) != 99 {
		t.Fatalf("members: %d %v", len(members), err)
	}
}

func TestSplitwiseFinalReviewUnicodeGroupCaseReimport(t *testing.T) {
	d := safetyDB(t)
	csv := "Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Demo taxi,General,100,INR,-50,50\n"
	for _, group := range []string{"Équipe", "équipe"} {
		entries, err := parser.ParseSplitwise(strings.NewReader(csv), group, "Sanjay")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
			t.Fatal(err)
		}
		if group == "Équipe" {
			if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: group, Name: "Asha Example", Pattern: "(?i)asha@fictional"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Group != "Équipe" || members[0].Pattern != "(?i)asha@fictional" {
		t.Fatalf("Unicode case-only reimport changed member identity: %+v", members)
	}
}

func TestSplitwiseFinalReviewManualCategories(t *testing.T) {
	for _, tc := range []struct {
		name     string
		linked   bool
		category string
	}{
		{name: "accountless explicit uncategorized"},
		{name: "linked explicit transfers", linked: true, category: models.CategoryTransfersID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := safetyDB(t)
			balances := "-50,50"
			if tc.linked {
				balances = "50,-50"
				a, err := d.GetOrCreateAccount("Demo bank", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
				if err != nil {
					t.Fatal(err)
				}
				tx := &models.Transaction{AccountID: a.ID, TxHash: "manual-category-demo", TxDate: "2026-09-01", RawNarration: "Demo taxi vendor", Amount: 100, TxType: models.TxTypeDebit}
				if _, err := d.UpsertTransaction(tx); err != nil {
					t.Fatal(err)
				}
			}
			entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Demo taxi,Fuel & Transport,100,INR,"+balances+"\n"), "Example", "Sanjay")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
				t.Fatal(err)
			}
			mappings, err := d.ListSplitwiseMappings()
			if err != nil || len(mappings) != 1 {
				t.Fatalf("mappings: %+v %v", mappings, err)
			}
			id := entries[0].ID
			if tc.linked {
				id = mappings[0].Bank.ID
			}
			if _, err := d.UpdateTransaction(id, models.UpdateTransactionRequest{CategoryID: &tc.category}); err != nil {
				t.Fatal(err)
			}
			if _, err := d.ReapplyRules(); err != nil {
				t.Fatal(err)
			}
			if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
				t.Fatal(err)
			}
			got, err := d.GetTransaction(id)
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsManualCategory {
				t.Fatal("manual category flag lost")
			}
			if tc.category == "" {
				if got.CategoryID != nil {
					t.Errorf("explicit Uncategorized reverted to %q", *got.CategoryID)
				}
			} else if got.CategoryID == nil || *got.CategoryID != tc.category {
				t.Errorf("manual Transfers overridden: %+v", got.CategoryID)
			}
			totals, err := d.GetAnalyticsOverview()
			if err != nil {
				t.Fatal(err)
			}
			want := float64(50)
			if tc.linked {
				want = 0
			}
			if totals.TotalExpense != want {
				t.Errorf("personal spending: got %v want %v", totals.TotalExpense, want)
			}
		})
	}
}

func TestSplitwiseFinalReviewCaseOnlyReimportKeepsMemberRules(t *testing.T) {
	d := safetyDB(t)
	first := "Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Demo taxi,Travel,100,INR,-50,50\n"
	entries, err := parser.ParseSplitwise(strings.NewReader(first), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "(?i)^asha@fictional$"}); err != nil {
		t.Fatal(err)
	}
	a, err := d.GetOrCreateAccount("Demo bank", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Two eligible movements prevent sole-candidate fallback from concealing a lost regex.
	paid := &models.Transaction{AccountID: a.ID, TxHash: "payment-demo", TxDate: "2026-09-02", RawNarration: "asha@fictional", CleanedPayee: "asha@fictional", Amount: 50, TxType: models.TxTypeDebit}
	other := &models.Transaction{AccountID: a.ID, TxHash: "unrelated-demo", TxDate: "2026-09-02", RawNarration: "Other vendor", CleanedPayee: "Other vendor", Amount: 50, TxType: models.TxTypeDebit}
	for _, tx := range []*models.Transaction{paid, other} {
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}
	changed := strings.ReplaceAll(first, "Sanjay Example,Asha Example", "SANJAY EXAMPLE,ASHA EXAMPLE") + "2026-09-02,Sanjay paid Asha,Payment,50,INR,50,-50\n"
	reimport, err := parser.ParseSplitwise(strings.NewReader(changed), "example", "SANJAY")
	if err != nil {
		t.Fatal(err)
	}
	if reimport[0].ID != entries[0].ID {
		t.Fatal("case-only export changed expense identity")
	}
	result, err := d.ImportSplitwiseAutomatically(reimport)
	if err != nil {
		t.Fatal(err)
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Errorf("case-only reimport duplicated member settings: %+v", members)
	}
	if len(members) == 1 && members[0].Pattern != "(?i)^asha@fictional$" {
		t.Errorf("saved regex lost: %+v", members[0])
	}
	if result.Transfers != 1 {
		t.Errorf("new CSV Payment did not use persisted member regex: %+v", result)
	}
	got, err := d.GetTransaction(paid.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SplitwiseKind == nil || *got.SplitwiseKind != "PAYMENT" {
		t.Error("intended bank repayment was not mapped")
	}
	untouched, err := d.GetTransaction(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if untouched.SplitwiseEntryID != nil {
		t.Error("unrelated bank movement was matched")
	}
}
