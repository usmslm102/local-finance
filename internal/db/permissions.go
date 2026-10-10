package db

import (
	"fmt"
	"os"
)

func protectDatabaseFiles(path string) error {
	for i, file := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Chmod(file, 0600); err != nil {
			if i > 0 && os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to protect database permissions: %w", err)
		}
	}
	return nil
}
