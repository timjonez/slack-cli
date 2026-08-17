package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

func (a *App) listenCmd() *cobra.Command {
	var (
		mentions bool
		thread   string
	)
	cmd := &cobra.Command{
		Use:   "listen [CHANNEL...]",
		Short: "Print inbound messages until interrupted",
		Long:  "Connect via Socket Mode and print messages. With no CHANNEL args, every conversation the bot is in is included. Status goes to stderr; the message stream goes to stdout.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withClient(true, func(c slackx.Client, _ config.Config) error {
				ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				defer stop()

				auth, err := c.AuthTest(ctx)
				if err != nil {
					return err
				}
				filter := slackx.Filter{
					Mentions:  mentions,
					ThreadTS:  thread,
					BotUserID: auth.UserID,
					BotID:     auth.BotID,
				}
				if len(args) > 0 {
					filter.Channels = make(map[string]struct{}, len(args))
					for _, ref := range args {
						ch, err := c.ResolveChannel(ctx, ref)
						if err != nil {
							return err
						}
						filter.Channels[ch.ID] = struct{}{}
						// IMs/MPIMs omit is_member; being able to resolve the DM means we are in it.
						if !ch.IsMember && !ch.IsIM && !ch.IsMPIM {
							fmt.Fprintf(a.Stderr, "warning: bot is not in %s; you will not receive messages until invited\n", ch.Display())
						}
					}
				}

				status := func(msg string) {
					if a.Quiet {
						return
					}
					fmt.Fprintln(a.Stderr, msg)
				}
				return c.Listen(ctx, filter, status, func(ev slackx.Event) error {
					if a.JSON {
						return a.writeJSON(ev)
					}
					fmt.Fprintln(a.Stdout, formatEvent(ev))
					return nil
				})
			})
		},
	}
	cmd.Flags().BoolVar(&mentions, "mentions", false, "only print messages that mention the bot")
	cmd.Flags().StringVar(&thread, "thread", "", "only print replies in this thread ts")
	return cmd
}
