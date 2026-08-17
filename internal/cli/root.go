package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

// Version is set at build time via -ldflags or defaults here.
var Version = "0.1.0"

// App holds shared CLI state.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	Quiet  bool
	JSON   bool

	loadConfig  func() (config.Config, error)
	newClient   func(cfg config.Config) (slackx.Client, error)
	stdinIsPipe func() bool
}

// NewApp constructs an App with process defaults.
func NewApp() *App {
	return &App{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
		loadConfig: config.Load,
		newClient: func(cfg config.Config) (slackx.Client, error) {
			return slackx.New(cfg.BotToken, cfg.AppToken), nil
		},
		stdinIsPipe: func() bool {
			st, err := os.Stdin.Stat()
			if err != nil {
				return false
			}
			return st.Mode()&os.ModeCharDevice == 0
		},
	}
}

// Execute runs the root command. Returns a process exit code.
func (a *App) Execute(args []string) int {
	root := a.rootCmd()
	root.SetOut(a.Stdout)
	root.SetErr(a.Stderr)
	if args != nil {
		root.SetArgs(args)
	}
	if err := root.Execute(); err != nil {
		if !errors.Is(err, errSilent) {
			if a.JSON {
				a.writeJSONError(err)
			} else {
				fmt.Fprintln(a.Stderr, err.Error())
			}
		}
		return 1
	}
	return 0
}

var errSilent = errors.New("silent")

func (a *App) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "slackcli",
		Short:         "Send, listen, and read as a Slack bot",
		Long:          "slackcli posts messages as a Slack bot, listens for channel replies over Socket Mode, and fetches messages by timestamp.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().BoolVarP(&a.Quiet, "quiet", "q", false, "minimal human output")
	root.PersistentFlags().BoolVar(&a.JSON, "json", false, "JSON output on stdout; errors as JSON on stderr")

	root.AddCommand(a.versionCmd())
	root.AddCommand(a.authCmd())
	root.AddCommand(a.whoamiCmd())
	root.AddCommand(a.sendCmd())
	root.AddCommand(a.getCmd())
	root.AddCommand(a.listenCmd())
	root.AddCommand(a.channelsCmd())
	root.AddCommand(a.joinCmd())
	root.AddCommand(a.completionCmd(root))
	return root
}

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.emitAlways(map[string]string{"version": Version}, func() {
				fmt.Fprintln(a.Stdout, Version)
			})
		},
	}
}

func (a *App) completionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish]",
		Short:     "Generate shell completion script",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(a.Stdout)
			case "zsh":
				return root.GenZshCompletion(a.Stdout)
			case "fish":
				return root.GenFishCompletion(a.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}

func (a *App) withClient(needApp bool, fn func(slackx.Client, config.Config) error) error {
	load := a.loadConfig
	if load == nil {
		load = config.Load
	}
	cfg, err := load()
	if err != nil {
		return err
	}
	if err := config.NeedBot(cfg); err != nil {
		return err
	}
	if needApp {
		if err := config.NeedApp(cfg); err != nil {
			return err
		}
	}
	newc := a.newClient
	if newc == nil {
		newc = func(cfg config.Config) (slackx.Client, error) {
			return slackx.New(cfg.BotToken, cfg.AppToken), nil
		}
	}
	c, err := newc(cfg)
	if err != nil {
		return err
	}
	return fn(c, cfg)
}
