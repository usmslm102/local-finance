package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type desktopActions struct {
	native       *application.App
	databasePath string
	dialogMu     sync.Mutex
}

func (d *desktopActions) OpenURL(url string) error { return d.native.Browser.OpenURL(url) }

func (d *desktopActions) SaveFile(name string, content io.Reader) error {
	d.dialogMu.Lock()
	defer d.dialogMu.Unlock()
	path, err := d.native.Dialog.SaveFile().SetFilename(name).PromptForSingleSelection()
	if err != nil || path == "" {
		return err
	}
	if err := protectActiveLedger(path, d.databasePath); err != nil {
		return err
	}
	// Complete the file before replacing the user's chosen destination.
	f, err := os.CreateTemp(filepath.Dir(path), ".localfinance-export-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := io.Copy(f, content); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func protectActiveLedger(destination, database string) error {
	databaseDir, err := filepath.EvalSymlinks(filepath.Dir(database))
	if err != nil {
		return err
	}
	database = filepath.Join(databaseDir, filepath.Base(database))
	dir, err := filepath.EvalSymlinks(filepath.Dir(destination))
	if err != nil {
		return err
	}
	destination = filepath.Join(dir, filepath.Base(destination))
	equal := func(a, b string) bool {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			return strings.EqualFold(a, b)
		}
		return a == b
	}
	for _, suffix := range []string{"", "-wal", "-shm", ".lock", ".desktop-port"} {
		if equal(destination, database+suffix) {
			return fmt.Errorf("choose an export destination outside the active ledger files")
		}
	}
	active, activeErr := os.Stat(database)
	target, targetErr := os.Stat(destination)
	if activeErr == nil && targetErr == nil && os.SameFile(active, target) {
		return fmt.Errorf("cannot export over the active ledger")
	}
	return nil
}
