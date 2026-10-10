package mcp

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Check the drive mapping before Lstat can contact a mapped network share.
func localCSVVolume(path string) bool {
	root, err := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if err != nil {
		return false
	}
	return localCSVDriveType(windows.GetDriveType(root))
}

func localCSVDriveType(kind uint32) bool {
	switch kind {
	case windows.DRIVE_FIXED, windows.DRIVE_REMOVABLE, windows.DRIVE_CDROM, windows.DRIVE_RAMDISK:
		return true
	default:
		return false
	}
}
