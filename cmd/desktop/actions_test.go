package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExportsCannotReplaceActiveLedger(t *testing.T) {
	dir := t.TempDir()
	database := filepath.Join(dir, "ledger.db")
	if err := os.WriteFile(database, []byte("ledger"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal", "-shm", ".lock", ".desktop-port"} {
		if err := protectActiveLedger(database+suffix, database); err == nil {
			t.Fatalf("allowed overwriting %s", suffix)
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		for _, suffix := range []string{"", "-wal", "-shm", ".lock", ".desktop-port"} {
			if err := protectActiveLedger(filepath.Join(dir, strings.ToUpper("ledger.db"+suffix)), database); err == nil {
				t.Fatalf("allowed case-variant ledger file: %s", suffix)
			}
		}
	}
	alias := filepath.Join(dir, "linked.db")
	if err := os.Link(database, alias); err == nil {
		if err := protectActiveLedger(alias, database); err == nil {
			t.Fatal("hard-link bypass")
		}
	}
	if err := protectActiveLedger(filepath.Join(dir, "backup.db"), database); err != nil {
		t.Fatal(err)
	}
}
