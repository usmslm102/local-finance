package mcp

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestCSVRejectsRemoteAndUnknownDriveTypes(t *testing.T) {
	for _, kind := range []uint32{windows.DRIVE_REMOTE, windows.DRIVE_UNKNOWN, windows.DRIVE_NO_ROOT_DIR} {
		if localCSVDriveType(kind) {
			t.Errorf("accepted nonlocal drive type %d", kind)
		}
	}
	if !localCSVDriveType(windows.DRIVE_FIXED) {
		t.Fatal("rejected fixed local drive")
	}
}
