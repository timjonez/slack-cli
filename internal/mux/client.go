package mux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/timjonez/slack-cli/internal/slackx"
)

const dialTimeout = 200 * time.Millisecond

// Subscribe connects to serve at socket and emits matching events until ctx is done.
func Subscribe(ctx context.Context, socket string, filter slackx.Filter, status func(string), emit func(slackx.Event) error) error {
	if status == nil {
		status = func(string) {}
	}
	if emit == nil {
		return errors.New("emit required")
	}
	if socket == "" {
		return errors.New("socket path required")
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return fmt.Errorf("dial serve at %s: %w", socket, err)
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	if err := writeFrame(conn, frameFromFilter(filter)); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	sc := newScanner(conn)
	for {
		f, err := readFrame(sc)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, io.EOF) {
				return errors.New("serve disconnected")
			}
			return err
		}
		switch f.Op {
		case opStatus:
			status(f.Msg)
		case opEvent:
			if f.Event == nil {
				continue
			}
			if err := emit(*f.Event); err != nil {
				return err
			}
		case opError:
			if f.Error == "" {
				return errors.New("serve error")
			}
			return errors.New(f.Error)
		default:
			// ignore unknown ops so older serves can add fields
		}
	}
}
