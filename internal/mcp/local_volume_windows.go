package mcp

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// Check the drive mapping before Lstat can contact a mapped network share.
func localStatementVolume(path string) bool {
	// Reject DOS device aliases (including names with extensions) before any open.
	relative := strings.TrimPrefix(filepath.Clean(path), filepath.VolumeName(path))
	for _, component := range strings.FieldsFunc(relative, func(r rune) bool { return r == '/' || r == '\\' }) {
		// Use the stem too so extension aliases are rejected consistently on
		// Windows 10 and 11, whose device-name normalization differs.
		stem, _, _ := strings.Cut(component, ".")
		stem = strings.TrimRight(stem, " ")
		if !filepath.IsLocal(component) || (stem != "" && !filepath.IsLocal(stem)) {
			return false
		}
	}
	root, err := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if err != nil {
		return false
	}
	return localStatementDriveType(windows.GetDriveType(root))
}

func localStatementDriveType(kind uint32) bool {
	switch kind {
	case windows.DRIVE_FIXED, windows.DRIVE_REMOVABLE, windows.DRIVE_CDROM, windows.DRIVE_RAMDISK:
		return true
	default:
		return false
	}
}
