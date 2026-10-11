package db

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"local-finance/internal/models"
)

func TestSavingsGoalLifecycleBackupExportAndReset(t *testing.T) {
	d, err := NewDB(filepath.Join(t.TempDir(), "planning.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	goal := models.SavingsGoal{Name: " Emergency fund ", TargetAmount: 300000, SavedAmount: 50000, MonthlyContribution: 10000, TargetDate: "2027-06-30"}
	if ok, err := d.SaveSavingsGoal(&goal, ""); err != nil || !ok {
		t.Fatalf("create: %v", err)
	}
	if goal.ID == "" || goal.Name != "Emergency fund" {
		t.Fatalf("invalid saved identity: %+v", goal)
	}
	exported, err := d.ExportAllDataJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(exported.Goals) != 1 {
		t.Fatal("goal missing from JSON export")
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err := d.BackupTo(backup); err != nil {
		t.Fatal(err)
	}
	goal.SavedAmount = 125000
	if ok, err := d.SaveSavingsGoal(&goal, goal.ID); err != nil || !ok {
		t.Fatalf("update: %v", err)
	}
	goals, err := d.ListSavingsGoals()
	if err != nil || goals[0].SavedAmount != 125000 {
		t.Fatalf("updated progress not persisted: %v", err)
	}
	if ok, err := d.SaveSavingsGoal(&goal, "missing"); err != nil || ok {
		t.Fatal("update unexpectedly created a missing goal")
	}
	if err := d.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	goals, err = d.ListSavingsGoals()
	if err != nil || len(goals) != 0 {
		t.Fatal("reset retained goal")
	}
	file, err := os.Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := d.RestoreFrom(file); err != nil {
		t.Fatal(err)
	}
	goals, err = d.ListSavingsGoals()
	if err != nil || len(goals) != 1 || goals[0].SavedAmount != 50000 {
		t.Fatalf("backup did not preserve goals: %v", err)
	}
	if ok, err := d.DeleteSavingsGoal(goal.ID); err != nil || !ok {
		t.Fatalf("delete: %v", err)
	}
	if ok, err := d.DeleteSavingsGoal(goal.ID); err != nil || ok {
		t.Fatal("missing goal deletion reported success")
	}
}

func TestSavingsGoalValidation(t *testing.T) {
	valid := models.SavingsGoal{Name: "Laptop", TargetAmount: 100000.01, SavedAmount: 0, MonthlyContribution: 5000.05, TargetDate: "2028-02-29"}
	for _, mutate := range []func(*models.SavingsGoal){
		func(g *models.SavingsGoal) { g.Name = "  " }, func(g *models.SavingsGoal) { g.TargetDate = "2027-02-29" }, func(g *models.SavingsGoal) { g.TargetDate = "0001-01-01" },
		func(g *models.SavingsGoal) { g.TargetAmount = 0 }, func(g *models.SavingsGoal) { g.TargetAmount = 1e12 + 1 }, func(g *models.SavingsGoal) { g.SavedAmount = -1 },
		func(g *models.SavingsGoal) { g.MonthlyContribution = math.NaN() }, func(g *models.SavingsGoal) { g.TargetAmount = math.Inf(1) }, func(g *models.SavingsGoal) { g.SavedAmount = 0.001 }, func(g *models.SavingsGoal) { g.SavedAmount = 0.0001 },
	} {
		value := valid
		mutate(&value)
		if err := validateGoal(&value); !errors.Is(err, ErrInvalidGoal) {
			t.Errorf("invalid goal accepted: %+v", value)
		}
	}
	if err := validateGoal(&valid); err != nil {
		t.Fatal(err)
	}
}

func TestDataHealthReviewFilterMatchesAggregate(t *testing.T) {
	d, err := NewDB(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	_, err = d.conn.Exec(`INSERT INTO accounts(id,bank_name,account_type,account_number_mask) VALUES('a','Bank','SAVINGS','1234'),('empty','Bank','SAVINGS','5678')`)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id, category       string
		transfer, excluded int
	}{
		{"missing", "", 0, 0}, {"others", "cat_others", 0, 0}, {"transfer", "cat_others", 1, 0}, {"excluded", "cat_others", 0, 1}, {"category-transfer", "cat_transfers", 0, 0}, {"categorized", "cat_food", 0, 0},
	} {
		_, err := d.conn.Exec(`INSERT INTO transactions(id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,reference_number,payment_mode,tx_type,amount,category_id,is_transfer,is_excluded,notes,tags) VALUES(?,'a',?,'2026-10-01','Test','Test','','OTHER','DEBIT',100,NULLIF(?,''),?,?,'','')`, row.id, row.id, row.category, row.transfer, row.excluded)
		if err != nil {
			t.Fatal(err)
		}
	}
	stats, err := d.GetDataHealthAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 || stats[0].ReviewCount != 2 || stats[1].ReviewCount != 0 {
		t.Fatalf("incorrect stats: %+v", stats)
	}
	rows, total, err := d.ListTransactions(TransactionFilter{ReviewNeeded: true, AccountID: "a", Limit: 50})
	if err != nil || total != 2 || len(rows) != 2 {
		t.Fatalf("filter differs: %d %v", total, err)
	}
}
