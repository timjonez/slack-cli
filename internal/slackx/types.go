package slackx

import (
	"context"
	"errors"
	"strings"

	"github.com/slack-go/slack"
)

// ErrNotFound means a channel or user could not be resolved.
var ErrNotFound = errors.New("not found")

// Client is the Slack surface the CLI uses. Tests swap in a fake.
type Client interface {
	AuthTest(ctx context.Context) (Auth, error)
	PostMessage(ctx context.Context, channelID, text, threadTS string) (Posted, error)
	Join(ctx context.Context, channelID string) error
	ResolveChannel(ctx context.Context, nameOrID string) (Channel, error)
	ListJoined(ctx context.Context) ([]Channel, error)
	UserName(ctx context.Context, userID string) (string, error)
	ChannelName(ctx context.Context, channelID string) (string, error)
	Listen(ctx context.Context, filter Filter, status func(string), emit func(Event) error) error
}

// Auth is the result of auth.test.
type Auth struct {
	URL    string `json:"url"`
	Team   string `json:"team"`
	TeamID string `json:"team_id"`
	User   string `json:"user"`
	UserID string `json:"user_id"`
	BotID  string `json:"bot_id"`
}

// Channel is a conversation the CLI can name and post to.
type Channel struct {
	ID        string
	Name      string // without '#'; empty for IMs
	IsIM      bool
	IsMPIM    bool
	IsPrivate bool
	IsMember  bool
	User      string // other user id for IMs
}

// Display is the human name: #eng, dm, or the raw id.
func (c Channel) Display() string {
	if c.IsIM {
		if c.Name != "" {
			return "dm:" + strings.TrimPrefix(c.Name, "@")
		}
		return "dm"
	}
	if c.Name != "" {
		return "#" + strings.TrimPrefix(c.Name, "#")
	}
	return c.ID
}

// Posted is a successful chat.postMessage.
type Posted struct {
	Channel string
	TS      string
	Text    string
}

// Event is one inbound message after name resolution.
type Event struct {
	Type        string `json:"type"`
	Channel     string `json:"channel"`
	ChannelName string `json:"channel_name"`
	User        string `json:"user"`
	UserName    string `json:"user_name"`
	TS          string `json:"ts"`
	ThreadTS    string `json:"thread_ts"`
	Text        string `json:"text"`
}

// Filter selects which inbound messages listen emits.
type Filter struct {
	// Channels, if non-empty, is the set of channel IDs to keep.
	Channels  map[string]struct{}
	ThreadTS  string
	Mentions  bool
	BotUserID string
	BotID     string
}

// Incoming is a raw Slack message/mention before name lookup.
type Incoming struct {
	Type     string
	Channel  string
	User     string
	BotID    string
	Username string
	TS       string
	ThreadTS string
	Text     string
	SubType  string
}

// SlackErrCode returns the Slack API error string, or "".
func SlackErrCode(err error) string {
	var se slack.SlackErrorResponse
	if errors.As(err, &se) {
		return se.Err
	}
	return ""
}

// IsNotInChannel reports the chat.postMessage / conversations.join membership error.
func IsNotInChannel(err error) bool {
	return SlackErrCode(err) == "not_in_channel"
}
