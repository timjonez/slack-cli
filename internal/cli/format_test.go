package cli

import (
	"strings"
	"testing"

	"github.com/timjonez/slack-cli/internal/slackx"
)

func TestFormatEvent(t *testing.T) {
	ev := slackx.Event{
		Channel:     "C1",
		ChannelName: "#eng",
		User:        "U1",
		UserName:    "alice",
		TS:          "1710000000.000200",
		Text:        "hello back",
	}
	got := formatEvent(ev)
	if !strings.Contains(got, "#eng") || !strings.Contains(got, "@alice") || !strings.Contains(got, "hello back") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "↳") {
		t.Fatalf("top-level should not mark thread: %q", got)
	}

	reply := ev
	reply.ThreadTS = "1710000000.000100"
	got = formatEvent(reply)
	if !strings.Contains(got, "↳") {
		t.Fatalf("reply should mark thread: %q", got)
	}

	parent := ev
	parent.ThreadTS = parent.TS
	got = formatEvent(parent)
	if strings.Contains(got, "↳") {
		t.Fatalf("parent should not mark thread: %q", got)
	}
}

func TestChannelKind(t *testing.T) {
	if channelKind(slackx.Channel{IsIM: true}) != "im" {
		t.Fatal("im")
	}
	if channelKind(slackx.Channel{IsPrivate: true}) != "private" {
		t.Fatal("private")
	}
	if channelKind(slackx.Channel{}) != "public" {
		t.Fatal("public")
	}
}
