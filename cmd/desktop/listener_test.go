package main

import (
	"path/filepath"
	"testing"
)

func TestDesktopOriginSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	first, err := listenDesktop(path)
	if err != nil {
		t.Fatal(err)
	}
	address := first.Addr().String()
	if other, err := listenDesktop(path); err == nil {
		other.Close()
		t.Fatal("silently changed origin when port was occupied")
	}
	first.Close()
	second, err := listenDesktop(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if second.Addr().String() != address {
		t.Fatal("desktop lost its persisted origin")
	}
}
