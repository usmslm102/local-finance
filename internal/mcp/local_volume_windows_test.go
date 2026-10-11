package mcp

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCSVRejectsRemoteAndUnknownDriveTypes(t *testing.T) {
	for _, kind := range []uint32{windows.DRIVE_REMOTE, windows.DRIVE_UNKNOWN, windows.DRIVE_NO_ROOT_DIR} {
		if localStatementDriveType(kind) {
			t.Errorf("accepted nonlocal drive type %d", kind)
		}
	}
	if !localStatementDriveType(windows.DRIVE_FIXED) {
		t.Fatal("rejected fixed local drive")
	}
}

func TestStatementPathsRejectDOSDeviceAliases(t *testing.T) {
	root := t.TempDir()
	for _, component := range []string{"NUL.csv", "CON.xlsx", "COM1.csv", "LPT9.xls", "AUX.csv", "PRN.csv", "COM¹.csv"} {
		path := filepath.Join(root, component)
		if validLocalStatementPath(path, []string{".csv", ".xlsx", ".xls"}) {
			t.Errorf("accepted DOS device %q", component)
		}
	}
	if validLocalStatementPath(filepath.Join(root, "NUL", "statement.csv"), []string{".csv"}) {
		t.Fatal("accepted DOS device ancestor")
	}
}
