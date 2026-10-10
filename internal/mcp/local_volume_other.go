//go:build !windows

package mcp

func localStatementVolume(string) bool { return true }
