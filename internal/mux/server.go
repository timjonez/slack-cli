// Package mux fans Slack Socket Mode events out to local listen clients
// over a unix socket so many processes can share one websocket.
package mux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/timjonez/slack-cli/internal/slackx"
)

// ErrAlreadyRunning means another serve holds the socket lock.
var ErrAlreadyRunning = errors.New("slackcli serve already running")

// Alive reports whether a serve is accepting connections on socket.
func Alive(socket string) bool {
	c, err := net.DialTimeout("unix", socket, dialTimeout)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// Run binds socket, holds a single Socket Mode connection, and fans events
// out to local listen subscribers. It returns when ctx is done or Slack
// listen ends.
func Run(ctx context.Context, socket string, c slackx.Client, status func(string)) error {
	if status == nil {
		status = func(string) {}
	}
	if socket == "" {
		return errors.New("socket path required")
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}

	lock, err := acquireLock(lockPath(socket))
	if err != nil {
		if errors.Is(err, ErrAlreadyRunning) {
			return fmt.Errorf("%w at %s", ErrAlreadyRunning, socket)
		}
		return err
	}
	defer lock.Close()

	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket: %w", err)
	}

	old := syscall.Umask(0o077)
	ln, err := net.Listen("unix", socket)
	syscall.Umask(old)
	if err != nil {
		return fmt.Errorf("listen %s: %w", socket, err)
	}
	defer ln.Close()
	defer os.Remove(socket)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	h := NewHub()
	defer h.Close()
	go acceptLoop(ctx, ln, h)

	auth, err := c.AuthTest(ctx)
	if err != nil {
		return err
	}
	slackStatus := func(msg string) {
		status(msg)
		h.SetStatus(msg)
	}
	return c.Listen(ctx, slackx.Filter{
		BotUserID: auth.UserID,
		BotID:     auth.BotID,
	}, slackStatus, func(ev slackx.Event) error {
		h.Broadcast(ev)
		return nil
	})
}

func acceptLoop(ctx context.Context, ln net.Listener, h *Hub) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleConn(ctx, conn, h)
	}
}

func handleConn(ctx context.Context, conn net.Conn, h *Hub) {
	defer conn.Close()

	sc := newScanner(conn)
	req, err := readFrame(sc)
	if err != nil {
		return
	}
	if req.Op != opSubscribe {
		_ = writeFrame(conn, frame{Op: opError, Error: "expected subscribe"})
		return
	}

	id, events, statuses, err := h.Subscribe(filterFromFrame(req))
	if err != nil {
		_ = writeFrame(conn, frame{Op: opError, Error: err.Error()})
		return
	}
	defer h.Unsubscribe(id)

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer cancel()
		_, _ = io.Copy(io.Discard, conn)
	}()

	for {
		select {
		case <-connCtx.Done():
			return
		case msg, ok := <-statuses:
			if !ok {
				return
			}
			if err := writeFrame(conn, frame{Op: opStatus, Msg: msg}); err != nil {
				return
			}
		case ev, ok := <-events:
			if !ok {
				return
			}
			event := ev
			if err := writeFrame(conn, frame{Op: opEvent, Event: &event}); err != nil {
				return
			}
		}
	}
}
