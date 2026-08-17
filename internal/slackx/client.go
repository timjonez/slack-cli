package slackx

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/slack-go/slack"
)

// API is the slack-go adapter.
type API struct {
	api *slack.Client

	users sync.Map // userID -> name
	chans sync.Map // channelID -> Channel
}

// New builds a Web API client. appToken may be empty for send-only commands.
func New(botToken, appToken string) *API {
	opts := []slack.Option{slack.OptionRetry(3)}
	if appToken != "" {
		opts = append(opts, slack.OptionAppLevelToken(appToken))
	}
	return &API{api: slack.New(botToken, opts...)}
}

func (a *API) AuthTest(ctx context.Context) (Auth, error) {
	r, err := a.api.AuthTestContext(ctx)
	if err != nil {
		return Auth{}, fmt.Errorf("auth.test: %w", err)
	}
	return Auth{
		URL:    r.URL,
		Team:   r.Team,
		TeamID: r.TeamID,
		User:   r.User,
		UserID: r.UserID,
		BotID:  r.BotID,
	}, nil
}

func (a *API) PostMessage(ctx context.Context, channelID, text, threadTS string) (Posted, error) {
	opts := []slack.MsgOption{slack.MsgOptionText(text, false)}
	if threadTS != "" {
		opts = append(opts, slack.MsgOptionTS(threadTS))
	}
	ch, ts, err := a.api.PostMessageContext(ctx, channelID, opts...)
	if err != nil {
		return Posted{}, err
	}
	return Posted{Channel: ch, TS: ts, Text: text}, nil
}

func (a *API) Join(ctx context.Context, channelID string) error {
	_, _, _, err := a.api.JoinConversationContext(ctx, channelID)
	if err != nil {
		return fmt.Errorf("join %s: %w", channelID, err)
	}
	return nil
}

func (a *API) ResolveChannel(ctx context.Context, nameOrID string) (Channel, error) {
	id, name := ParseChannelRef(nameOrID)
	if id == "" && name == "" {
		return Channel{}, fmt.Errorf("%w: empty channel", ErrNotFound)
	}
	if id != "" {
		return a.lookupID(ctx, id)
	}
	return a.lookupName(ctx, name)
}

func (a *API) lookupID(ctx context.Context, id string) (Channel, error) {
	if v, ok := a.chans.Load(id); ok {
		return v.(Channel), nil
	}
	info, err := a.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{ChannelID: id})
	if err != nil {
		return Channel{}, fmt.Errorf("channel %s: %w", id, err)
	}
	ch := fromSlackChannel(*info)
	a.chans.Store(ch.ID, ch)
	return ch, nil
}

func (a *API) lookupName(ctx context.Context, name string) (Channel, error) {
	name = strings.ToLower(name)
	joined, err := a.ListJoined(ctx)
	if err != nil {
		return Channel{}, err
	}
	for _, ch := range joined {
		if strings.EqualFold(ch.Name, name) {
			return ch, nil
		}
	}
	all, err := a.listAll(ctx, []string{"public_channel", "private_channel"})
	if err != nil {
		return Channel{}, err
	}
	for _, ch := range all {
		if strings.EqualFold(ch.Name, name) {
			return ch, nil
		}
	}
	return Channel{}, fmt.Errorf("%w: channel %q", ErrNotFound, name)
}

func (a *API) ListJoined(ctx context.Context) ([]Channel, error) {
	raw, err := a.listForUser(ctx, []string{"public_channel", "private_channel", "im", "mpim"})
	if err != nil {
		return nil, err
	}
	out := make([]Channel, 0, len(raw))
	for _, c := range raw {
		ch := fromSlackChannel(c)
		if ch.IsIM {
			if n, err := a.UserName(ctx, ch.User); err == nil && n != "" {
				ch.Name = n
			}
		}
		a.chans.Store(ch.ID, ch)
		out = append(out, ch)
	}
	return out, nil
}

func (a *API) UserName(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", nil
	}
	if v, ok := a.users.Load(userID); ok {
		return v.(string), nil
	}
	u, err := a.api.GetUserInfoContext(ctx, userID)
	if err != nil {
		return "", err
	}
	name := displayName(u)
	a.users.Store(userID, name)
	return name, nil
}

func (a *API) ChannelName(ctx context.Context, channelID string) (string, error) {
	ch, err := a.lookupID(ctx, channelID)
	if err != nil {
		return channelID, err
	}
	return ch.Display(), nil
}

func (a *API) listForUser(ctx context.Context, types []string) ([]slack.Channel, error) {
	var all []slack.Channel
	cursor := ""
	for {
		page, next, err := a.api.GetConversationsForUserContext(ctx, &slack.GetConversationsForUserParameters{
			Types:           types,
			Limit:           200,
			Cursor:          cursor,
			ExcludeArchived: true,
		})
		if err != nil {
			return nil, fmt.Errorf("conversations: %w", err)
		}
		all = append(all, page...)
		if next == "" {
			return all, nil
		}
		cursor = next
	}
}

func (a *API) listAll(ctx context.Context, types []string) ([]Channel, error) {
	var all []Channel
	cursor := ""
	for {
		page, next, err := a.api.GetConversationsContext(ctx, &slack.GetConversationsParameters{
			Types:           types,
			Limit:           200,
			Cursor:          cursor,
			ExcludeArchived: true,
		})
		if err != nil {
			return nil, fmt.Errorf("conversations.list: %w", err)
		}
		for _, c := range page {
			ch := fromSlackChannel(c)
			a.chans.Store(ch.ID, ch)
			all = append(all, ch)
		}
		if next == "" {
			return all, nil
		}
		cursor = next
	}
}

func fromSlackChannel(c slack.Channel) Channel {
	return Channel{
		ID:        c.ID,
		Name:      c.Name,
		IsIM:      c.IsIM,
		IsMPIM:    c.IsMpIM,
		IsPrivate: c.IsPrivate,
		IsMember:  c.IsMember,
		User:      c.User,
	}
}

func displayName(u *slack.User) string {
	if u == nil {
		return ""
	}
	if n := strings.TrimSpace(u.Profile.DisplayName); n != "" {
		return n
	}
	if n := strings.TrimSpace(u.Profile.RealName); n != "" {
		return n
	}
	if n := strings.TrimSpace(u.RealName); n != "" {
		return n
	}
	if n := strings.TrimSpace(u.Name); n != "" {
		return n
	}
	return u.ID
}
