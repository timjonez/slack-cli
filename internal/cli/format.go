package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/timjonez/slack-cli/internal/slackx"
)

func formatEvent(ev slackx.Event) string {
	when := slackTSTime(ev.TS).Local().Format("15:04:05")
	ch := ev.ChannelName
	if ch == "" {
		ch = ev.Channel
	}
	user := ev.UserName
	if user == "" {
		user = ev.User
	}
	if user != "" && !strings.HasPrefix(user, "@") {
		user = "@" + user
	}
	text := ev.Text
	if ev.ThreadTS != "" && ev.ThreadTS != ev.TS {
		text = "↳  " + text
	}
	return fmt.Sprintf("%s  %s  %s  %s", when, ch, user, text)
}

func slackTSTime(ts string) time.Time {
	sec, _, _ := strings.Cut(ts, ".")
	n, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		return time.Now()
	}
	return time.Unix(n, 0)
}

func channelKind(ch slackx.Channel) string {
	switch {
	case ch.IsIM:
		return "im"
	case ch.IsMPIM:
		return "mpim"
	case ch.IsPrivate:
		return "private"
	default:
		return "public"
	}
}
