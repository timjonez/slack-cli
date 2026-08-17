package slackx

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// GetMessage fetches one message by timestamp. Thread replies are not in
// conversations.history, so a miss falls back to conversations.replies.
func (a *API) GetMessage(ctx context.Context, channelID, ts string) (Event, error) {
	if ts == "" {
		return Event{}, fmt.Errorf("%w: empty message ts", ErrNotFound)
	}
	resp, err := a.api.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Latest:    ts,
		Oldest:    ts,
		Inclusive: true,
		Limit:     1,
	})
	if err != nil {
		return Event{}, mapHistoryErr(fmt.Errorf("conversations.history: %w", err), ts)
	}
	if len(resp.Messages) == 1 && resp.Messages[0].Timestamp == ts {
		return a.eventFromMsg(ctx, channelID, resp.Messages[0])
	}
	msgs, err := a.replies(ctx, channelID, ts)
	if err != nil {
		return Event{}, mapHistoryErr(err, ts)
	}
	if m, ok := pickMessage(msgs, ts); ok {
		return a.eventFromMsg(ctx, channelID, m)
	}
	return Event{}, fmt.Errorf("%w: message %s", ErrNotFound, ts)
}

// GetThread fetches a parent and its replies. TS may be the parent or a reply.
func (a *API) GetThread(ctx context.Context, channelID, ts string) ([]Event, error) {
	if ts == "" {
		return nil, fmt.Errorf("%w: empty message ts", ErrNotFound)
	}
	msgs, err := a.replies(ctx, channelID, ts)
	if err != nil {
		return nil, mapHistoryErr(err, ts)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("%w: message %s", ErrNotFound, ts)
	}
	out := make([]Event, 0, len(msgs))
	for _, m := range msgs {
		ev, err := a.eventFromMsg(ctx, channelID, m)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func (a *API) replies(ctx context.Context, channelID, ts string) ([]slack.Message, error) {
	var all []slack.Message
	cursor := ""
	for {
		page, hasMore, next, err := a.api.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: ts,
			Cursor:    cursor,
			Limit:     200,
		})
		if err != nil {
			return nil, fmt.Errorf("conversations.replies: %w", err)
		}
		all = append(all, page...)
		if !hasMore || next == "" {
			return all, nil
		}
		cursor = next
	}
}

func (a *API) eventFromMsg(ctx context.Context, channelID string, m slack.Message) (Event, error) {
	return a.resolveEvent(ctx, Incoming{
		Type:     "message",
		Channel:  channelID,
		User:     m.User,
		BotID:    m.BotID,
		Username: m.Username,
		TS:       m.Timestamp,
		ThreadTS: m.ThreadTimestamp,
		Text:     m.Text,
		SubType:  m.SubType,
	})
}

func pickMessage(msgs []slack.Message, ts string) (slack.Message, bool) {
	for _, m := range msgs {
		if m.Timestamp == ts {
			return m, true
		}
	}
	return slack.Message{}, false
}

func mapHistoryErr(err error, ts string) error {
	switch SlackErrCode(err) {
	case "thread_not_found", "message_not_found":
		return fmt.Errorf("%w: message %s", ErrNotFound, ts)
	default:
		return err
	}
}
