package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Preserve the localhost origin across launches so browser-stored preferences
// (including the release-check opt-out) survive desktop restarts and upgrades.
// The first launch still asks the OS to choose a free port.
func listenDesktop(databasePath string) (net.Listener, error) {
	portFile := databasePath + ".desktop-port"
	port := 0
	content, err := os.ReadFile(portFile)
	if err == nil {
		port, err = strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid desktop port file: %s", portFile)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		if port == 0 {
			return nil, fmt.Errorf("cannot allocate a desktop loopback port: %w", err)
		}
		return nil, fmt.Errorf("desktop port %d is unavailable; close the process using it and reopen LocalFinance, or remove %s to choose a new port (this resets browser-stored preferences, including the update-check opt-out): %w", port, portFile, err)
	}
	if port == 0 {
		file, err := os.CreateTemp(filepath.Dir(portFile), ".desktop-port-*")
		if err != nil {
			listener.Close()
			return nil, err
		}
		defer os.Remove(file.Name())
		_, writeErr := fmt.Fprintln(file, listener.Addr().(*net.TCPAddr).Port)
		closeErr := file.Close()
		if writeErr != nil {
			listener.Close()
			return nil, writeErr
		}
		if closeErr != nil {
			listener.Close()
			return nil, closeErr
		}
		if err := os.Rename(file.Name(), portFile); err != nil {
			listener.Close()
			return nil, err
		}
	}
	return listener, nil
}
