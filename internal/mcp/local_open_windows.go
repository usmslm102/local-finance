package mcp

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Keep ancestor handles open without write/delete sharing so checked directories
// cannot be renamed/replaced during traversal. OPEN_REPARSE_POINT opens links
// themselves, allowing rejection without contacting their targets.
func openStatementWithoutSymlinks(path string) (*os.File, error) {
	var paths []string
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		paths = append(paths, current)
		if filepath.Dir(current) == current {
			break
		}
	}
	var parents []windows.Handle
	defer func() {
		for _, handle := range parents {
			windows.CloseHandle(handle)
		}
	}()
	for i := len(paths) - 1; i >= 0; i-- {
		name, err := windows.UTF16PtrFromString(paths[i])
		if err != nil {
			return nil, err
		}
		handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
		if err != nil {
			return nil, err
		}
		var info windows.ByHandleFileInformation
		err = windows.GetFileInformationByHandle(handle, &info)
		if err != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || (i > 0 && info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0) {
			windows.CloseHandle(handle)
			return nil, errors.New("symlinks and invalid ancestors are unsupported")
		}
		if i == 0 {
			return os.NewFile(uintptr(handle), path), nil
		}
		parents = append(parents, handle)
	}
	return nil, errors.New("invalid statement path")
}
