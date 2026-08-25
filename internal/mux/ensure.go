package mux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const ensureTimeout = 8 * time.Second

// Ensure makes sure a serve is accepting on socket, starting one if needed.
func Ensure(ctx context.Context, socket string, start func(string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if Alive(socket) {
		return nil
	}
	if start == nil {
		return fmt.Errorf("serve is not running at %s", socket)
	}
	if err := start(socket); err != nil {
		if Alive(socket) {
			return nil
		}
		return err
	}

	deadline := time.Now().Add(ensureTimeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if Alive(socket) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("serve did not start at %s (see %s)", socket, LogPath(socket))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// StartDetached launches `slackcli serve --socket PATH` in the background.
func StartDetached(socket string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	logf, err := os.OpenFile(LogPath(socket), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("serve log: %w", err)
	}
	defer logf.Close()

	cmd := exec.Command(exe, "serve", "--socket", socket)
	cmd.Env = os.Environ()
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start serve: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("detach serve: %w", err)
	}
	return nil
}
