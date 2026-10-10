package mcp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func openLocalStatementFile(path string, maxSize int64, extensions []string) (*os.File, error) {
	if !validLocalStatementPath(path, extensions) {
		return nil, errors.New("provide an absolute local path with a supported statement extension on the LocalFinance host")
	}
	path = filepath.Clean(path)
	if err := checkStatementPathComponents(path); err != nil {
		return nil, errors.New("statement must be accessible locally; symlinks are unsupported")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSize {
		return nil, errors.New("statement must be a regular file within the tool's file-size limit")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("unable to open statement on the LocalFinance host")
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSize {
		file.Close()
		return nil, errors.New("statement must be a regular file within the tool's file-size limit")
	}
	return file, nil
}

// Check Windows separator semantics even on other hosts before any filesystem call.
func validLocalStatementPath(path string, extensions []string) bool {
	supported := false
	for _, extension := range extensions {
		if strings.EqualFold(filepath.Ext(path), extension) {
			supported = true
			break
		}
	}
	normalized := strings.ReplaceAll(path, "\\", "/")
	return filepath.IsAbs(path) && supported && !strings.HasPrefix(normalized, "//") && !strings.Contains(strings.TrimPrefix(path, filepath.VolumeName(path)), ":") && localStatementVolume(path)
}

// Inspect ancestors from the root so directory symlinks/reparse points are rejected
// before a later Lstat or Open could follow them onto another filesystem.
func checkStatementPathComponents(path string) error {
	clean := filepath.Clean(path)
	var paths []string
	for current := clean; ; current = filepath.Dir(current) {
		paths = append(paths, current)
		if parent := filepath.Dir(current); parent == current {
			break
		}
	}
	for i := len(paths) - 1; i >= 0; i-- {
		info, err := os.Lstat(paths[i])
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlinks unsupported")
		}
		if i > 0 && !info.IsDir() {
			return errors.New("Statement parent must be a directory")
		}
	}
	return nil
}
