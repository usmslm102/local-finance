//go:build !windows

package mcp

func localCSVVolume(string) bool { return true }
