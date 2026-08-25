package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds Slack tokens. Empty fields are unset.
type Config struct {
	BotToken string `json:"bot_token"`
	AppToken string `json:"app_token"`
}

var (
	ErrMissingBotToken = errors.New("bot token required (SLACK_BOT_TOKEN or slackcli auth set --bot-token)")
	ErrMissingAppToken = errors.New("app token required for listen/serve (SLACK_APP_TOKEN or slackcli auth set --app-token)")
	ErrBadBotPrefix    = errors.New("bot token must start with xoxb-")
	ErrBadAppPrefix    = errors.New("app token must start with xapp-")
)

// Path is the resolved config file path.
// SLACKCLI_CONFIG overrides; otherwise $XDG_CONFIG_HOME/slackcli/config.json
// or ~/.config/slackcli/config.json.
func Path() string {
	if p := os.Getenv("SLACKCLI_CONFIG"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", "slackcli", "config.json")
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "slackcli", "config.json")
}

// Load merges environment tokens over the config file.
// A missing file is not an error.
func Load() (Config, error) {
	cfg, err := ReadFile(Path())
	if err != nil {
		return Config{}, err
	}
	if v := firstEnv("SLACKCLI_BOT_TOKEN"); v != "" {
		cfg.BotToken = v
	}
	if v := firstEnv("SLACKCLI_APP_TOKEN"); v != "" {
		cfg.AppToken = v
	}
	// Unqualified Slack names win — same convention as Bolt / slack-go examples.
	if v := firstEnv("SLACK_BOT_TOKEN"); v != "" {
		cfg.BotToken = v
	}
	if v := firstEnv("SLACK_APP_TOKEN"); v != "" {
		cfg.AppToken = v
	}
	return cfg, nil
}

// ReadFile loads tokens from path only (no env overlay). Missing file yields empty Config.
func ReadFile(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes cfg to Path() with mode 0600.
func Save(cfg Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// NeedBot checks a bot token is present and well-prefixed.
func NeedBot(cfg Config) error {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return ErrMissingBotToken
	}
	if !strings.HasPrefix(cfg.BotToken, "xoxb-") {
		return ErrBadBotPrefix
	}
	return nil
}

// NeedApp checks an app-level token is present and well-prefixed.
func NeedApp(cfg Config) error {
	if strings.TrimSpace(cfg.AppToken) == "" {
		return ErrMissingAppToken
	}
	if !strings.HasPrefix(cfg.AppToken, "xapp-") {
		return ErrBadAppPrefix
	}
	return nil
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
