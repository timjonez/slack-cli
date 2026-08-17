package slackx

import "testing"

func TestKeep(t *testing.T) {
	base := Incoming{
		Type:    "message",
		Channel: "C1",
		User:    "U1",
		TS:      "1.0",
		Text:    "hello",
	}
	f := Filter{BotUserID: "UBOT", BotID: "BBOT"}

	if !Keep(base, f) {
		t.Fatal("plain message should keep")
	}
	ownUser := base
	ownUser.User = "UBOT"
	if Keep(ownUser, f) {
		t.Fatal("own user should skip")
	}
	ownBot := base
	ownBot.BotID = "BBOT"
	if Keep(ownBot, f) {
		t.Fatal("own bot id should skip")
	}
	changed := base
	changed.SubType = "message_changed"
	if Keep(changed, f) {
		t.Fatal("message_changed should skip")
	}
	join := base
	join.SubType = "channel_join"
	if Keep(join, f) {
		t.Fatal("channel_join should skip")
	}
	threadOk := base
	threadOk.ThreadTS = "9.0"
	if !Keep(threadOk, Filter{}) {
		t.Fatal("unfiltered thread should keep")
	}
	if Keep(threadOk, Filter{ThreadTS: "8.0"}) {
		t.Fatal("other thread should skip")
	}
	if !Keep(threadOk, Filter{ThreadTS: "9.0"}) {
		t.Fatal("matching thread_ts should keep")
	}
	parent := base
	parent.TS = "9.0"
	if !Keep(parent, Filter{ThreadTS: "9.0"}) {
		t.Fatal("parent ts should keep")
	}
	wrongCh := base
	if Keep(wrongCh, Filter{Channels: map[string]struct{}{"C2": {}}}) {
		t.Fatal("other channel should skip")
	}
	if !Keep(base, Filter{Channels: map[string]struct{}{"C1": {}}}) {
		t.Fatal("matching channel should keep")
	}

	mentions := Filter{Mentions: true, BotUserID: "UBOT"}
	if Keep(base, mentions) {
		t.Fatal("plain text should skip on --mentions")
	}
	hit := base
	hit.Text = "hey <@UBOT> hi"
	if !Keep(hit, mentions) {
		t.Fatal("mention in text should keep")
	}
	app := base
	app.Type = "mention"
	if !Keep(app, mentions) {
		t.Fatal("app_mention type should keep")
	}

	empty := Incoming{User: "U1"}
	if Keep(empty, Filter{}) {
		t.Fatal("missing channel/ts should skip")
	}

	broadcast := base
	broadcast.SubType = "thread_broadcast"
	if !Keep(broadcast, Filter{}) {
		t.Fatal("thread_broadcast should keep")
	}
}

func TestSeenSetDedupe(t *testing.T) {
	s := newSeen()
	if !s.add("C1", "1.0") {
		t.Fatal("first should accept")
	}
	if s.add("C1", "1.0") {
		t.Fatal("duplicate should reject")
	}
	if !s.add("C1", "2.0") {
		t.Fatal("new ts should accept")
	}
}
