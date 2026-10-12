package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Exercise the OS lock across processes, including inherited directory aliases.
func TestDatabaseOwnership(t *testing.T) {
	if path := os.Getenv("LOCALFINANCE_LOCK_TEST_PATH"); path != "" {
		if _, err := Open(path, nil); err == nil {
			os.Exit(2)
		}
		return
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.db")
	owner, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	child := exec.Command(os.Args[0], "-test.run=^TestDatabaseOwnership$")
	child.Env = append(os.Environ(), "LOCALFINANCE_LOCK_TEST_PATH="+path)
	if result, err := child.CombinedOutput(); err != nil {
		t.Fatalf("child acquired owned database: %s: %v", result, err)
	}
	aliasDir := filepath.Join(dir, "alias")
	if err := os.Symlink(dir, aliasDir); err == nil {
		if duplicate, err := Open(filepath.Join(aliasDir, "ledger.db"), nil); err == nil {
			duplicate.Close()
			t.Fatal("directory alias bypassed ownership")
		}
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path, nil)
	if err != nil {
		t.Fatalf("lock survived close: %v", err)
	}
	defer reopened.Close()
}

func TestDatabaseLockRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.db")
	if err := os.Symlink(target, path+".lock"); err != nil {
		t.Skip(err)
	}
	if owner, err := Open(path, nil); err == nil {
		owner.Close()
		t.Fatal("accepted symlink lock file")
	}
}
