package mux

import (
	"path/filepath"
	"testing"
)

func TestSocketPathEnvOverride(t *testing.T) {
	t.Setenv("SLACKCLI_SOCKET", "/tmp/custom.sock")
	if got := SocketPath(); got != "/tmp/custom.sock" {
		t.Fatalf("got %q", got)
	}
	if got := ResolveSocket(""); got != "/tmp/custom.sock" {
		t.Fatalf("resolve empty %q", got)
	}
	if got := ResolveSocket("/other.sock"); got != "/other.sock" {
		t.Fatalf("resolve flag %q", got)
	}
}

func TestSocketPathDefaultUsesRuntimeDir(t *testing.T) {
	t.Setenv("SLACKCLI_SOCKET", "")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	got := SocketPath()
	want := filepath.Join("/run/user/1000", "slackcli", "slackcli.sock")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestLogPath(t *testing.T) {
	got := LogPath("/run/user/1000/slackcli/slackcli.sock")
	want := "/run/user/1000/slackcli/serve.log"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
