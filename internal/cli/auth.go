package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
)

func (a *App) authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Slack tokens",
	}
	cmd.AddCommand(a.authPathCmd())
	cmd.AddCommand(a.authSetCmd())
	return cmd
}

func (a *App) authPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := config.Path()
			return a.emitAlways(map[string]string{"path": p}, func() {
				fmt.Fprintln(a.Stdout, p)
			})
		},
	}
}

func (a *App) authSetCmd() *cobra.Command {
	var bot, app string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Save bot and/or app tokens to the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if bot == "" && app == "" {
				return fmt.Errorf("provide --bot-token and/or --app-token")
			}
			fileCfg, err := config.ReadFile(config.Path())
			if err != nil {
				return err
			}
			if bot != "" {
				fileCfg.BotToken = bot
			}
			if app != "" {
				fileCfg.AppToken = app
			}
			if fileCfg.BotToken != "" {
				if err := config.NeedBot(fileCfg); err != nil {
					return err
				}
			}
			if fileCfg.AppToken != "" {
				if err := config.NeedApp(fileCfg); err != nil {
					return err
				}
			}
			if err := config.Save(fileCfg); err != nil {
				return err
			}
			return a.emit(map[string]string{"path": config.Path()}, true, func() {
				fmt.Fprintf(a.Stdout, "saved %s\n", config.Path())
			})
		},
	}
	cmd.Flags().StringVar(&bot, "bot-token", "", "bot token (xoxb-...)")
	cmd.Flags().StringVar(&app, "app-token", "", "app-level token (xapp-...)")
	return cmd
}
