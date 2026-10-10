//go:build linux || darwin

package mcp

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
)

// Traverse using directory descriptors: a concurrent rename cannot redirect a
// checked ancestor. NOFOLLOW rejects links at every step; NONBLOCK prevents a
// replaced FIFO from blocking before the caller validates the opened file.
func openStatementWithoutSymlinks(path string) (*os.File, error) {
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(path), "/"), "/")
	for i, part := range parts {
		flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
		if i < len(parts)-1 {
			flags |= unix.O_DIRECTORY
		}
		next, openErr := unix.Openat(fd, part, flags, 0)
		unix.Close(fd)
		if openErr != nil {
			return nil, openErr
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), path), nil
}
