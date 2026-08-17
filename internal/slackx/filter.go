package slackx

import "strings"

// junkSubtypes are Slack message subtypes that are not chat.
var junkSubtypes = map[string]struct{}{
	"message_changed":             {},
	"message_deleted":             {},
	"channel_join":                {},
	"channel_leave":               {},
	"channel_topic":               {},
	"channel_purpose":             {},
	"channel_name":                {},
	"channel_archive":             {},
	"channel_unarchive":           {},
	"group_join":                  {},
	"group_leave":                 {},
	"group_topic":                 {},
	"group_purpose":               {},
	"group_name":                  {},
	"group_archive":               {},
	"group_unarchive":             {},
	"pinned_item":                 {},
	"unpinned_item":               {},
	"ekm_access_denied":           {},
	"channel_posting_permissions": {},
}

// Keep reports whether in should be printed for f.
func Keep(in Incoming, f Filter) bool {
	if in.Channel == "" || in.TS == "" {
		return false
	}
	if in.SubType != "" {
		if _, junk := junkSubtypes[in.SubType]; junk {
			return false
		}
	}
	if f.BotID != "" && in.BotID == f.BotID {
		return false
	}
	if f.BotUserID != "" && in.User == f.BotUserID {
		return false
	}
	if len(f.Channels) > 0 {
		if _, ok := f.Channels[in.Channel]; !ok {
			return false
		}
	}
	if f.ThreadTS != "" {
		if in.ThreadTS != f.ThreadTS && in.TS != f.ThreadTS {
			return false
		}
	}
	if f.Mentions {
		if in.Type == "mention" {
			return true
		}
		if f.BotUserID != "" && strings.Contains(in.Text, "<@"+f.BotUserID+">") {
			return true
		}
		return false
	}
	return true
}
