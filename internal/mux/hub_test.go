package mux

import (
	"testing"
	"time"

	"github.com/timjonez/slack-cli/internal/slackx"
)

func TestHubFanoutByFilter(t *testing.T) {
	h := NewHub()
	defer h.Close()

	_, allCh, _, err := h.Subscribe(slackx.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	_, threadCh, _, err := h.Subscribe(slackx.Filter{ThreadTS: "9.0"})
	if err != nil {
		t.Fatal(err)
	}
	_, mentionsCh, _, err := h.Subscribe(slackx.Filter{Mentions: true, BotUserID: "UBOT"})
	if err != nil {
		t.Fatal(err)
	}

	plain := slackx.Event{Type: "message", Channel: "C1", User: "U1", TS: "1.0", Text: "hello"}
	reply := slackx.Event{Type: "message", Channel: "C1", User: "U1", TS: "2.0", ThreadTS: "9.0", Text: "reply"}
	mention := slackx.Event{Type: "mention", Channel: "C1", User: "U1", TS: "3.0", Text: "hey <@UBOT>"}

	h.Broadcast(plain)
	h.Broadcast(reply)
	h.Broadcast(mention)

	gotAll := recvN(t, allCh, 3)
	if gotAll[0].TS != "1.0" || gotAll[1].TS != "2.0" || gotAll[2].TS != "3.0" {
		t.Fatalf("all=%+v", gotAll)
	}
	gotThread := recvN(t, threadCh, 1)
	if gotThread[0].TS != "2.0" {
		t.Fatalf("thread=%+v", gotThread)
	}
	gotMention := recvN(t, mentionsCh, 1)
	if gotMention[0].TS != "3.0" {
		t.Fatalf("mentions=%+v", gotMention)
	}
	assertNone(t, threadCh)
	assertNone(t, mentionsCh)
}

func TestHubChannelFilter(t *testing.T) {
	h := NewHub()
	defer h.Close()
	_, ch, _, err := h.Subscribe(slackx.Filter{Channels: map[string]struct{}{"C1": {}}})
	if err != nil {
		t.Fatal(err)
	}
	h.Broadcast(slackx.Event{Type: "message", Channel: "C2", User: "U1", TS: "1.0", Text: "nope"})
	h.Broadcast(slackx.Event{Type: "message", Channel: "C1", User: "U1", TS: "2.0", Text: "yes"})
	got := recvN(t, ch, 1)
	if got[0].TS != "2.0" {
		t.Fatalf("%+v", got)
	}
	assertNone(t, ch)
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	h := NewHub()
	defer h.Close()
	id, ch, _, err := h.Subscribe(slackx.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	h.Unsubscribe(id)
	h.Unsubscribe(id)
	h.Broadcast(slackx.Event{Type: "message", Channel: "C1", User: "U1", TS: "1.0", Text: "x"})
	if _, ok := <-ch; ok {
		t.Fatal("expected closed channel")
	}
}

func TestHubStatusReplay(t *testing.T) {
	h := NewHub()
	defer h.Close()
	h.SetStatus("connected")
	_, _, status, err := h.Subscribe(slackx.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-status:
		if msg != "connected" {
			t.Fatalf("status %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("no replayed status")
	}
}

func TestHubSubscribeAfterClose(t *testing.T) {
	h := NewHub()
	h.Close()
	if _, _, _, err := h.Subscribe(slackx.Filter{}); err == nil {
		t.Fatal("expected error")
	}
}

func recvN(t *testing.T, ch <-chan slackx.Event, n int) []slackx.Event {
	t.Helper()
	out := make([]slackx.Event, 0, n)
	for len(out) < n {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("channel closed after %d, want %d", len(out), n)
			}
			out = append(out, ev)
		case <-time.After(time.Second):
			t.Fatalf("timeout after %d events, want %d", len(out), n)
		}
	}
	return out
}

func assertNone(t *testing.T, ch <-chan slackx.Event) {
	t.Helper()
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event %+v", ev)
	case <-time.After(20 * time.Millisecond):
	}
}
