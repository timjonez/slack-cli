package mux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SocketPath is the unix socket slackcli serve binds and listen dials.
// SLACKCLI_SOCKET overrides; otherwise $XDG_RUNTIME_DIR/slackcli/slackcli.sock
// (or a uid-scoped temp dir when XDG_RUNTIME_DIR is unset).
func SocketPath() string {
	if p := strings.TrimSpace(os.Getenv("SLACKCLI_SOCKET")); p != "" {
		return p
	}
	dir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if dir == "" {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("slackcli-%d", os.Getuid()))
	}
	return filepath.Join(dir, "slackcli", "slackcli.sock")
}

// ResolveSocket prefers an explicit --socket flag, then SocketPath.
func ResolveSocket(flag string) string {
	if p := strings.TrimSpace(flag); p != "" {
		return p
	}
	return SocketPath()
}

// LogPath is where a detached serve writes stdout/stderr.
func LogPath(socket string) string {
	return filepath.Join(filepath.Dir(socket), "serve.log")
}

func lockPath(socket string) string {
	return socket + ".lock"
}
