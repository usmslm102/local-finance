//go:build linux || darwin

package mcp

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStatementOpenRejectsFIFOWithoutBlocking(t *testing.T) {
	// Resolve platform temp-root aliases; the test concerns the leaf, not /var links.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "statement.csv")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		file, err := openLocalStatementFile(path, 1024, []string{".csv"})
		if file != nil {
			file.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("accepted FIFO")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("file opening blocked on FIFO")
	}
}

func TestStatementOpenCannotFollowConcurrentLeafReplacement(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "statement.csv")
	secret := filepath.Join(dir, "private.csv")
	if err := os.WriteFile(secret, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			os.Remove(path)
			os.Symlink(secret, path)
			os.Remove(path)
			os.WriteFile(path, []byte("statement"), 0600)
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	for i := 0; i < 500; i++ {
		file, err := openLocalStatementFile(path, 1024, []string{".csv"})
		if err != nil {
			continue
		}
		data := make([]byte, 32)
		n, _ := file.Read(data)
		file.Close()
		if string(data[:n]) == "private" {
			t.Fatal("followed concurrently replaced symlink")
		}
	}
}
