package updater

import (
	"context"
	"testing"
)

func TestDesktopInstallationSelection(t *testing.T) {
	assets := []GitHubAsset{
		{Name: "LocalFinance.exe"},
		{Name: "local-finance-darwin-universal.tar.gz"},
		{Name: "checksums.txt"},
		{Name: "LocalFinance-Desktop.dmg"},
		{Name: "LocalFinance-Desktop-windows-arm64-setup.exe"},
		{Name: "local-finance-desktop-linux-amd64.deb"},
	}
	for _, tc := range []struct{ os, arch, want string }{
		{"darwin", "arm64", "LocalFinance-Desktop.dmg"},
		{"darwin", "amd64", "LocalFinance-Desktop.dmg"},
		{"windows", "arm64", "LocalFinance-Desktop-windows-arm64-setup.exe"},
		{"linux", "amd64", "local-finance-desktop-linux-amd64.deb"},
		{"windows", "amd64", ""},
	} {
		asset, checksums := SelectDesktopAsset(assets, tc.os, tc.arch)
		if checksums == nil {
			t.Fatal("missing manifest")
		}
		if tc.want == "" {
			if asset != nil {
				t.Fatal("fell back to server executable")
			}
			continue
		}
		if asset == nil || asset.Name != tc.want {
			t.Fatalf("%s/%s: wrong installer %v", tc.os, tc.arch, asset)
		}
	}
}

func TestDesktopNeverReplacesExecutable(t *testing.T) {
	s := NewDesktopService()
	info, err := s.CheckForUpdate(context.Background(), false, true)
	if err != nil || info.CanAutoUpdate || info.AutoUpdateError == "" {
		t.Fatalf("unsafe offline desktop policy: %+v %v", info, err)
	}
	// Must refuse before network requests or executable/filesystem mutation.
	s.client = nil
	if _, err := s.ApplyUpdate(context.Background(), nil); err == nil {
		t.Fatal("desktop attempted executable replacement")
	}
}
