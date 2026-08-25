package mux

import (
	"errors"
	"sync"

	"github.com/timjonez/slack-cli/internal/slackx"
)

const subBuffer = 64

var errHubClosed = errors.New("serve stopped")

type subscriber struct {
	filter slackx.Filter
	events chan slackx.Event
	status chan string
}

// Hub fans resolved Slack events out to local subscribers.
type Hub struct {
	mu         sync.Mutex
	next       int
	subs       map[int]*subscriber
	lastStatus string
	closed     bool
}

// NewHub builds an empty fan-out hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[int]*subscriber)}
}

// Subscribe registers a filter. The caller must Unsubscribe.
func (h *Hub) Subscribe(f slackx.Filter) (int, <-chan slackx.Event, <-chan string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return 0, nil, nil, errHubClosed
	}
	id := h.next
	h.next++
	s := &subscriber{
		filter: f,
		events: make(chan slackx.Event, subBuffer),
		status: make(chan string, 8),
	}
	h.subs[id] = s
	if h.lastStatus != "" {
		s.status <- h.lastStatus
	}
	return id, s.events, s.status, nil
}

// Unsubscribe drops a subscriber. Safe to call twice.
func (h *Hub) Unsubscribe(id int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.subs[id]
	if !ok {
		return
	}
	delete(h.subs, id)
	close(s.events)
	close(s.status)
}

// SetStatus records the latest Slack connection status and notifies subscribers.
func (h *Hub) SetStatus(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.lastStatus = msg
	for _, s := range h.subs {
		select {
		case s.status <- msg:
		default:
		}
	}
}

// Broadcast sends ev to every subscriber whose filter matches.
func (h *Hub) Broadcast(ev slackx.Event) {
	in := incomingFromEvent(ev)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	for _, s := range h.subs {
		if !slackx.Keep(in, s.filter) {
			continue
		}
		select {
		case s.events <- ev:
		default:
		}
	}
}

// Close unregisters every subscriber. Further Subscribe calls fail.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for id, s := range h.subs {
		close(s.events)
		close(s.status)
		delete(h.subs, id)
	}
}

func incomingFromEvent(ev slackx.Event) slackx.Incoming {
	return slackx.Incoming{
		Type:     ev.Type,
		Channel:  ev.Channel,
		User:     ev.User,
		TS:       ev.TS,
		ThreadTS: ev.ThreadTS,
		Text:     ev.Text,
	}
}
