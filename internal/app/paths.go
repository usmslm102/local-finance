package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultDatabasePath preserves the portable application's existing ledger.
func DefaultDatabasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find data directory: %w", err)
	}
	return filepath.Join(home, ".localfinance", "local_finance.db"), nil
}

// CanonicalDatabasePath resolves directory and file aliases before taking a lock.
func CanonicalDatabasePath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	path = filepath.Join(dir, filepath.Base(path))
	resolved, err := filepath.EvalSymlinks(path)
	if os.IsNotExist(err) {
		return path, nil
	}
	return resolved, err
}
