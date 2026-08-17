package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

func (a *App) joinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "join CHANNEL",
		Short: "Join a public channel as the bot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withClient(false, func(c slackx.Client, _ config.Config) error {
				ch, err := c.ResolveChannel(context.Background(), args[0])
				if err != nil {
					return err
				}
				if err := c.Join(context.Background(), ch.ID); err != nil {
					return err
				}
				return a.emit(map[string]string{
					"joined":  ch.Display(),
					"channel": ch.ID,
				}, true, func() {
					fmt.Fprintf(a.Stdout, "joined %s\n", ch.Display())
				})
			})
		},
	}
}
