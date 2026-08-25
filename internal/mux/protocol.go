package mux

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/timjonez/slack-cli/internal/slackx"
)

const (
	opSubscribe = "subscribe"
	opStatus    = "status"
	opEvent     = "event"
	opError     = "error"

	maxFrame = 1 << 20
)

// frame is one JSONL object on the unix socket.
type frame struct {
	Op        string        `json:"op"`
	Channels  []string      `json:"channels,omitempty"`
	ThreadTS  string        `json:"thread_ts,omitempty"`
	Mentions  bool          `json:"mentions,omitempty"`
	BotUserID string        `json:"bot_user_id,omitempty"`
	Msg       string        `json:"msg,omitempty"`
	Event     *slackx.Event `json:"event,omitempty"`
	Error     string        `json:"error,omitempty"`
}

func newScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxFrame)
	return sc
}

func readFrame(sc *bufio.Scanner) (frame, error) {
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return frame{}, err
		}
		return frame{}, io.EOF
	}
	var f frame
	if err := json.Unmarshal(sc.Bytes(), &f); err != nil {
		return frame{}, fmt.Errorf("bad frame: %w", err)
	}
	return f, nil
}

func writeFrame(w io.Writer, f frame) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(f)
}

func frameFromFilter(f slackx.Filter) frame {
	return frame{
		Op:        opSubscribe,
		Channels:  channelIDs(f),
		ThreadTS:  f.ThreadTS,
		Mentions:  f.Mentions,
		BotUserID: f.BotUserID,
	}
}

func filterFromFrame(f frame) slackx.Filter {
	out := slackx.Filter{
		ThreadTS:  f.ThreadTS,
		Mentions:  f.Mentions,
		BotUserID: f.BotUserID,
	}
	if len(f.Channels) > 0 {
		out.Channels = make(map[string]struct{}, len(f.Channels))
		for _, id := range f.Channels {
			if id == "" {
				continue
			}
			out.Channels[id] = struct{}{}
		}
	}
	return out
}

func channelIDs(f slackx.Filter) []string {
	if len(f.Channels) == 0 {
		return nil
	}
	ids := make([]string, 0, len(f.Channels))
	for id := range f.Channels {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
