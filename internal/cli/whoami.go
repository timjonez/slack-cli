package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

func (a *App) whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the authenticated bot identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withClient(false, func(c slackx.Client, _ config.Config) error {
				auth, err := c.AuthTest(context.Background())
				if err != nil {
					return err
				}
				return a.emitAlways(auth, func() {
					fmt.Fprintf(a.Stdout, "team: %s (%s)\n", auth.Team, auth.TeamID)
					fmt.Fprintf(a.Stdout, "user: %s (%s)\n", auth.User, auth.UserID)
					if auth.BotID != "" {
						fmt.Fprintf(a.Stdout, "bot:  %s\n", auth.BotID)
					}
					if auth.URL != "" {
						fmt.Fprintf(a.Stdout, "url:  %s\n", auth.URL)
					}
				})
			})
		},
	}
}
