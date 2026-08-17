package slackx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// Listen connects via Socket Mode and emits matching messages until ctx is done.
func (a *API) Listen(ctx context.Context, filter Filter, status func(string), emit func(Event) error) error {
	if status == nil {
		status = func(string) {}
	}
	sm := socketmode.New(a.api)
	seen := newSeen()

	errCh := make(chan error, 1)
	go func() {
		errCh <- sm.RunContext(ctx)
	}()

	events := sm.Events
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		case err := <-errCh:
			if ctx.Err() != nil {
				return nil
			}
			if err == nil {
				return errors.New("socket mode connection closed")
			}
			return err
		case evt, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if err := a.handleSocketEvent(ctx, sm, evt, filter, seen, status, emit); err != nil {
				return err
			}
		}
	}
}

func (a *API) handleSocketEvent(ctx context.Context, sm *socketmode.Client, evt socketmode.Event, filter Filter, seen *seenSet, status func(string), emit func(Event) error) error {
	switch evt.Type {
	case socketmode.EventTypeConnecting:
		status("connecting to Slack...")
	case socketmode.EventTypeConnectionError:
		status("connection error (retrying)")
	case socketmode.EventTypeConnected:
		status("connected")
	case socketmode.EventTypeInvalidAuth:
		return errors.New("invalid Slack app token")
	case socketmode.EventTypeEventsAPI:
		if evt.Request != nil {
			_ = sm.Ack(*evt.Request)
		}
		apiEvt, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return nil
		}
		in, ok := incomingFrom(apiEvt)
		if !ok || !Keep(in, filter) {
			return nil
		}
		if !seen.add(in.Channel, in.TS) {
			return nil
		}
		ev, err := a.resolveEvent(ctx, in)
		if err != nil {
			status(fmt.Sprintf("resolve event: %v", err))
			// still emit with raw ids
			ev = Event{
				Type:        in.Type,
				Channel:     in.Channel,
				ChannelName: in.Channel,
				User:        in.User,
				UserName:    firstNonEmpty(in.Username, in.User),
				TS:          in.TS,
				ThreadTS:    in.ThreadTS,
				Text:        in.Text,
			}
		}
		return emit(ev)
	}
	return nil
}

func incomingFrom(apiEvt slackevents.EventsAPIEvent) (Incoming, bool) {
	if apiEvt.Type != slackevents.CallbackEvent {
		return Incoming{}, false
	}
	switch ev := apiEvt.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		return Incoming{
			Type:     "message",
			Channel:  ev.Channel,
			User:     ev.User,
			BotID:    ev.BotID,
			Username: ev.Username,
			TS:       ev.TimeStamp,
			ThreadTS: ev.ThreadTimeStamp,
			Text:     ev.Text,
			SubType:  ev.SubType,
		}, true
	case *slackevents.AppMentionEvent:
		return Incoming{
			Type:     "mention",
			Channel:  ev.Channel,
			User:     ev.User,
			BotID:    ev.BotID,
			TS:       ev.TimeStamp,
			ThreadTS: ev.ThreadTimeStamp,
			Text:     ev.Text,
		}, true
	default:
		return Incoming{}, false
	}
}

func (a *API) resolveEvent(ctx context.Context, in Incoming) (Event, error) {
	chName, err := a.ChannelName(ctx, in.Channel)
	if err != nil {
		chName = in.Channel
	}
	userName := in.Username
	if in.User != "" {
		if n, err := a.UserName(ctx, in.User); err == nil && n != "" {
			userName = n
		}
	}
	if userName == "" {
		userName = in.User
	}
	typ := in.Type
	if typ == "" {
		typ = "message"
	}
	return Event{
		Type:        typ,
		Channel:     in.Channel,
		ChannelName: chName,
		User:        in.User,
		UserName:    userName,
		TS:          in.TS,
		ThreadTS:    in.ThreadTS,
		Text:        in.Text,
	}, nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

type seenSet struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func newSeen() *seenSet {
	return &seenSet{m: make(map[string]time.Time)}
}

func (s *seenSet) add(channel, ts string) bool {
	key := channel + "\t" + ts
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[key]; ok {
		return false
	}
	now := time.Now()
	s.m[key] = now
	if len(s.m) > 256 {
		cutoff := now.Add(-2 * time.Minute)
		for k, t := range s.m {
			if t.Before(cutoff) {
				delete(s.m, k)
			}
		}
	}
	return true
}
