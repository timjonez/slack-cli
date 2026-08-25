package mux

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureNoopIfAlive(t *testing.T) {
	socket := testSocket(t)
	startDummy(t, socket)
	started := 0
	if err := Ensure(context.Background(), socket, func(string) error {
		started++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if started != 0 {
		t.Fatalf("start called %d times", started)
	}
}

func TestEnsureStartsAndWaits(t *testing.T) {
	socket := testSocket(t)
	ctx := context.Background()
	if err := Ensure(ctx, socket, func(s string) error {
		go func() {
			time.Sleep(80 * time.Millisecond)
			startDummy(t, s)
		}()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !Alive(socket) {
		t.Fatal("expected alive")
	}
}

func TestEnsureNilStart(t *testing.T) {
	socket := testSocket(t)
	err := Ensure(context.Background(), socket, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureStartErrorIgnoredIfAlive(t *testing.T) {
	socket := testSocket(t)
	err := Ensure(context.Background(), socket, func(s string) error {
		startDummy(t, s)
		return errors.New("already started")
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnsureTimeout(t *testing.T) {
	socket := testSocket(t)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err := Ensure(ctx, socket, func(string) error { return nil })
	if err == nil {
		t.Fatal("expected timeout")
	}
}

func testSocket(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "s.sock")
}

func startDummy(t *testing.T, socket string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
		_ = os.Remove(socket)
	})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
}
