package slackx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/slack-go/slack"
)

func TestPickMessage(t *testing.T) {
	msgs := []slack.Message{
		{Msg: slack.Msg{Timestamp: "1.0", Text: "parent"}},
		{Msg: slack.Msg{Timestamp: "2.0", ThreadTimestamp: "1.0", Text: "reply"}},
	}
	got, ok := pickMessage(msgs, "2.0")
	if !ok || got.Text != "reply" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	if _, ok := pickMessage(msgs, "9.0"); ok {
		t.Fatal("expected miss")
	}
}

func TestMapHistoryErr(t *testing.T) {
	wrapped := fmt.Errorf("conversations.replies: %w", slack.SlackErrorResponse{Err: "thread_not_found"})
	err := mapHistoryErr(wrapped, "1.0")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	plain := errors.New("boom")
	if mapHistoryErr(plain, "1.0") != plain {
		t.Fatal("should pass through")
	}
}
