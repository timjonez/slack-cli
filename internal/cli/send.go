package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

func (a *App) sendCmd() *cobra.Command {
	var thread string
	cmd := &cobra.Command{
		Use:   "send CHANNEL [TEXT...]",
		Short: "Post a message as the bot",
		Long:  "Post TEXT to CHANNEL as the Slack bot. CHANNEL is #name or a C/D/G id. If TEXT is omitted or '-', the message is read from stdin.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := a.messageText(args[1:])
			if err != nil {
				return err
			}
			return a.withClient(false, func(c slackx.Client, _ config.Config) error {
				ch, err := c.ResolveChannel(context.Background(), args[0])
				if err != nil {
					return err
				}
				posted, err := c.PostMessage(context.Background(), ch.ID, text, thread)
				if slackx.IsNotInChannel(err) {
					if jerr := c.Join(context.Background(), ch.ID); jerr != nil {
						return fmt.Errorf("not in %s and join failed: %w", ch.Display(), jerr)
					}
					posted, err = c.PostMessage(context.Background(), ch.ID, text, thread)
				}
				if err != nil {
					return err
				}
				return a.emitAlways(map[string]string{
					"channel": posted.Channel,
					"ts":      posted.TS,
					"text":    posted.Text,
				}, func() {
					fmt.Fprintf(a.Stdout, "%s  %s\n", ch.Display(), posted.TS)
				})
			})
		},
	}
	cmd.Flags().StringVar(&thread, "thread", "", "parent message ts to reply in a thread")
	return cmd
}

func (a *App) messageText(args []string) (string, error) {
	fromStdin := len(args) == 0 || (len(args) == 1 && args[0] == "-")
	if fromStdin {
		if !a.isPipe() {
			if len(args) == 1 && args[0] == "-" {
				return "", fmt.Errorf("stdin is not a pipe")
			}
			return "", fmt.Errorf("message text required")
		}
		b, err := io.ReadAll(a.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		text := strings.TrimRight(string(b), "\n")
		if text == "" {
			return "", fmt.Errorf("message text required")
		}
		return text, nil
	}
	return strings.Join(args, " "), nil
}

func (a *App) isPipe() bool {
	if a.stdinIsPipe != nil {
		return a.stdinIsPipe()
	}
	return false
}
