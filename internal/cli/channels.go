package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

type jsonChannel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Private bool   `json:"private"`
	Member  bool   `json:"member"`
}

func (a *App) channelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "channels",
		Short: "List conversations the bot is in",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withClient(false, func(c slackx.Client, _ config.Config) error {
				chs, err := c.ListJoined(context.Background())
				if err != nil {
					return err
				}
				out := make([]jsonChannel, 0, len(chs))
				for _, ch := range chs {
					out = append(out, jsonChannel{
						ID:      ch.ID,
						Name:    ch.Display(),
						Kind:    channelKind(ch),
						Private: ch.IsPrivate || ch.IsIM || ch.IsMPIM,
						Member:  ch.IsMember,
					})
				}
				return a.emitAlways(out, func() {
					for _, ch := range out {
						fmt.Fprintf(a.Stdout, "%-16s  %s  %s\n", ch.Name, ch.ID, ch.Kind)
					}
				})
			})
		},
	}
}
