package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

func (a *App) getCmd() *cobra.Command {
	var replies bool
	cmd := &cobra.Command{
		Use:   "get CHANNEL TS",
		Short: "Fetch a message by timestamp",
		Long:  "Fetch the message at TS in CHANNEL. TS may be a parent or a reply timestamp. --replies returns the parent plus the rest of the thread.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withClient(false, func(c slackx.Client, _ config.Config) error {
				ch, err := c.ResolveChannel(context.Background(), args[0])
				if err != nil {
					return err
				}
				ts := args[1]
				if replies {
					msgs, err := c.GetThread(context.Background(), ch.ID, ts)
					if err != nil {
						return err
					}
					return a.emitAlways(msgs, func() {
						for _, ev := range msgs {
							if ev.ChannelName == "" {
								ev.ChannelName = ch.Display()
							}
							fmt.Fprintln(a.Stdout, formatEvent(ev))
						}
					})
				}
				ev, err := c.GetMessage(context.Background(), ch.ID, ts)
				if err != nil {
					return err
				}
				if ev.ChannelName == "" {
					ev.ChannelName = ch.Display()
				}
				return a.emitAlways(ev, func() {
					fmt.Fprintln(a.Stdout, formatEvent(ev))
				})
			})
		},
	}
	cmd.Flags().BoolVar(&replies, "replies", false, "include the rest of the thread")
	return cmd
}
