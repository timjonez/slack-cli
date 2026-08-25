package mux

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/timjonez/slack-cli/internal/slackx"
)

type streamClient struct {
	auth    slackx.Auth
	authErr error
	events  chan slackx.Event
	started chan struct{}
}

func (s *streamClient) AuthTest(ctx context.Context) (slackx.Auth, error) {
	return s.auth, s.authErr
}
func (s *streamClient) PostMessage(context.Context, string, string, string) (slackx.Posted, error) {
	return slackx.Posted{}, errors.New("unused")
}
func (s *streamClient) GetMessage(context.Context, string, string) (slackx.Event, error) {
	return slackx.Event{}, errors.New("unused")
}
func (s *streamClient) GetThread(context.Context, string, string) ([]slackx.Event, error) {
	return nil, errors.New("unused")
}
func (s *streamClient) Join(context.Context, string) error { return errors.New("unused") }
func (s *streamClient) ResolveChannel(context.Context, string) (slackx.Channel, error) {
	return slackx.Channel{}, errors.New("unused")
}
func (s *streamClient) ListJoined(context.Context) ([]slackx.Channel, error) {
	return nil, errors.New("unused")
}
func (s *streamClient) UserName(context.Context, string) (string, error) { return "", nil }
func (s *streamClient) ChannelName(context.Context, string) (string, error) {
	return "", nil
}
func (s *streamClient) Listen(ctx context.Context, filter slackx.Filter, status func(string), emit func(slackx.Event) error) error {
	if status != nil {
		status("connected")
	}
	if s.started != nil {
		close(s.started)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-s.events:
			if !ok {
				return nil
			}
			if err := emit(ev); err != nil {
				return err
			}
		}
	}
}

func TestRunFansOutToSubscribers(t *testing.T) {
	socket := testSocket(t)
	client := &streamClient{
		auth:    slackx.Auth{UserID: "UBOT", BotID: "BBOT"},
		events:  make(chan slackx.Event, 8),
		started: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- Run(ctx, socket, client, nil) }()

	waitAlive(t, socket)
	<-client.started

	evA := make(chan slackx.Event, 4)
	evB := make(chan slackx.Event, 4)
	subA, cancelA := subscribeAsync(t, socket, slackx.Filter{ThreadTS: "9.0"}, evA)
	subB, cancelB := subscribeAsync(t, socket, slackx.Filter{}, evB)
	defer cancelA()
	defer cancelB()

	waitStatus(t, subA)
	waitStatus(t, subB)

	client.events <- slackx.Event{Type: "message", Channel: "C1", User: "U1", TS: "1.0", Text: "plain"}
	client.events <- slackx.Event{Type: "message", Channel: "C1", User: "U1", TS: "2.0", ThreadTS: "9.0", Text: "reply"}

	gotA := recvN(t, evA, 1)
	if gotA[0].TS != "2.0" {
		t.Fatalf("thread sub %+v", gotA)
	}
	gotB := recvN(t, evB, 2)
	if gotB[0].TS != "1.0" || gotB[1].TS != "2.0" {
		t.Fatalf("all sub %+v", gotB)
	}
	assertNone(t, evA)

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop")
	}
}

func TestRunSecondInstanceAlreadyRunning(t *testing.T) {
	socket := testSocket(t)
	client := &streamClient{
		auth:    slackx.Auth{UserID: "UBOT"},
		events:  make(chan slackx.Event),
		started: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 2)
	go func() { errCh <- Run(ctx, socket, client, nil) }()
	waitAlive(t, socket)
	<-client.started

	err := Run(ctx, socket, client, nil)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("got %v", err)
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop")
	}
}

func TestSubscribeCancelledReturnsNil(t *testing.T) {
	socket := testSocket(t)
	client := &streamClient{
		auth:    slackx.Auth{UserID: "UBOT"},
		events:  make(chan slackx.Event),
		started: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = Run(ctx, socket, client, nil) }()
	waitAlive(t, socket)
	<-client.started

	subCtx, subCancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Subscribe(subCtx, socket, slackx.Filter{}, func(string) {}, func(slackx.Event) error { return nil })
	}()
	time.Sleep(50 * time.Millisecond)
	subCancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe did not return")
	}
}

func subscribeAsync(t *testing.T, socket string, filter slackx.Filter, out chan slackx.Event) (chan string, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	status := make(chan string, 8)
	t.Cleanup(cancel)
	go func() {
		_ = Subscribe(ctx, socket, filter, func(msg string) {
			select {
			case status <- msg:
			default:
			}
		}, func(ev slackx.Event) error {
			out <- ev
			return nil
		})
	}()
	return status, cancel
}

func waitStatus(t *testing.T, status <-chan string) {
	t.Helper()
	select {
	case <-status:
	case <-time.After(2 * time.Second):
		t.Fatal("no status")
	}
}

func waitAlive(t *testing.T, socket string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if Alive(socket) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("serve not alive at %s", socket)
}
