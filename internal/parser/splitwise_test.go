package parser

import (
	"os"
	"strings"
	"testing"
)

func TestSplitwiseExport(t *testing.T) {
	data, err := os.ReadFile("../../samples/splitwise/fictional-group.csv")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := ParseSplitwise(strings.NewReader("\ufeff"+string(data)), "Example", "sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 7 || entries[0].Person != "Sanjay Example" || entries[0].Share != 10000 || entries[1].Share != 5000 || entries[2].Kind != "PAYMENT" || entries[3].Net != -8000 || entries[4].Share != 0 {
		t.Fatalf("incorrect export interpretation: %+v", entries)
	}
	for _, e := range entries {
		if e.Status != "REVIEW" {
			t.Fatal("net balances cannot prove paid amounts")
		}
	}
	reimport, err := ParseSplitwise(strings.NewReader(strings.ReplaceAll(string(data), "\n", "\r\n")), "example", "Sanjay Example")
	if err != nil {
		t.Fatal(err)
	}
	for i := range entries {
		if entries[i].ID != reimport[i].ID {
			t.Fatal("re-upload identity changed")
		}
	}
}

func TestSplitwiseValidationAndRepeatedExpenses(t *testing.T) {
	header := "Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n"
	row := "2026-09-01,Example,Dining,100.00,INR,50.00,-50.00\n"
	entries, err := ParseSplitwise(strings.NewReader(header+row+row), "Group", "Sanjay")
	if err != nil || len(entries) != 2 || entries[0].ID == entries[1].ID {
		t.Fatalf("repeated real expenses lost: %v", err)
	}
	for _, input := range []string{
		strings.Replace(header+row, "INR", "USD", 1),
		strings.Replace(header+row, "100.00", "NaN", 1),
		strings.Replace(header+row, "100.00", "1e2", 1),
		strings.Replace(header+row, "100.00", "100.001", 1),
		strings.Replace(header+row, "-50.00", "-49.99", 1),
		strings.Replace(header+row, "2026-09-01", "2026-02-30", 1),
		header + "2026-09-01,truncated\n",
		strings.Replace(header+row, "Asha Example", "Sanjay Other", 1),
	} {
		if _, err := ParseSplitwise(strings.NewReader(input), "Group", "Sanjay"); err == nil {
			t.Fatalf("accepted invalid input %q", input)
		}
	}
	if _, err := ParseSplitwise(strings.NewReader(header+row), "Group", "Unknown"); err == nil {
		t.Fatal("accepted unknown member")
	}
	if _, err := ParseSplitwise(strings.NewReader(header+row), "", "Sanjay"); err == nil {
		t.Fatal("accepted empty group")
	}
	reordered := "Date,Description,Category,Cost,Currency,New Member,Asha Example,Sanjay Example\n2026-09-01,Example,Dining,100.00,INR,0.00,-50.00,50.00\n"
	got, err := ParseSplitwise(strings.NewReader(reordered), "Group", "Sanjay")
	if err != nil || got[0].ID != entries[0].ID {
		t.Fatalf("member reorder or zero-balance addition changed identity: %v", err)
	}
}
