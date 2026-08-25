package mux

import (
	"reflect"
	"testing"

	"github.com/timjonez/slack-cli/internal/slackx"
)

func TestFilterFrameRoundTrip(t *testing.T) {
	in := slackx.Filter{
		Channels:  map[string]struct{}{"C2": {}, "C1": {}},
		ThreadTS:  "9.0",
		Mentions:  true,
		BotUserID: "UBOT",
	}
	got := filterFromFrame(frameFromFilter(in))
	if got.ThreadTS != in.ThreadTS || got.Mentions != in.Mentions || got.BotUserID != in.BotUserID {
		t.Fatalf("got %+v", got)
	}
	if !reflect.DeepEqual(got.Channels, in.Channels) {
		t.Fatalf("channels %+v", got.Channels)
	}
	ids := channelIDs(in)
	if len(ids) != 2 || ids[0] != "C1" || ids[1] != "C2" {
		t.Fatalf("ids %v", ids)
	}
}
