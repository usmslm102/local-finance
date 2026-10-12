package app

import (
	"fmt"
	"os"
)

// Keep the lock file in place: unlinking it lets a second process lock a new inode.
type databaseLock struct{ file *os.File }

func acquireDatabaseLock(path string) (*databaseLock, error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(lockPath)
	actual, statErr := f.Stat()
	if err != nil || statErr != nil || !info.Mode().IsRegular() || !os.SameFile(info, actual) {
		f.Close()
		return nil, fmt.Errorf("database lock is not a regular file: %s", lockPath)
	}
	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("ledger is already open or cannot be locked; quit the other LocalFinance desktop/server before opening %s: %w", path, err)
	}
	return &databaseLock{file: f}, nil
}

func (l *databaseLock) Close() { _ = l.file.Close() }
