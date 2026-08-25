package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/mux"
	"github.com/timjonez/slack-cli/internal/slackx"
)

func (a *App) serveCmd() *cobra.Command {
	var socket string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the shared Socket Mode daemon",
		Long:  "Hold one Slack Socket Mode connection and fan inbound messages out to local slackcli listen clients over a unix socket. listen starts this automatically if needed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			sock := mux.ResolveSocket(socket)
			if mux.Alive(sock) {
				return fmt.Errorf("%w at %s", mux.ErrAlreadyRunning, sock)
			}
			return a.withClient(true, func(c slackx.Client, _ config.Config) error {
				ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				defer stop()
				status := func(msg string) {
					if a.Quiet {
						return
					}
					fmt.Fprintln(a.Stderr, msg)
				}
				if !a.Quiet {
					fmt.Fprintf(a.Stderr, "serving on %s\n", sock)
				}
				return mux.Run(ctx, sock, c, status)
			})
		},
	}
	cmd.Flags().StringVar(&socket, "socket", "", "unix socket path (default: $SLACKCLI_SOCKET or $XDG_RUNTIME_DIR/slackcli/slackcli.sock)")
	return cmd
}
