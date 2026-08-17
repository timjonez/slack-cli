package slackx

import (
	"regexp"
	"strings"
)

// slackID matches a Slack conversation id (channel, group, IM, or org-wide).
var slackID = regexp.MustCompile(`^[CDGW][A-Z0-9]{8,}$`)

// ParseChannelRef classifies a user-supplied channel as an id or a name.
// Leading '#' is stripped from names. Empty input is returned as a name.
func ParseChannelRef(s string) (id string, name string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if slackID.MatchString(s) {
		return s, ""
	}
	return "", strings.TrimPrefix(s, "#")
}
