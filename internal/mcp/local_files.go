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
	file, err := openStatementWithoutSymlinks(path)
	if err != nil {
		return nil, errors.New("statement must be accessible locally; symlinks and special files are unsupported")
	}

	info, err := file.Stat()
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
