package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	writeCfg(t, path, Config{BotToken: "xoxb-file", AppToken: "xapp-file"})
	t.Setenv("SLACKCLI_CONFIG", path)
	t.Setenv("SLACKCLI_BOT_TOKEN", "xoxb-cli")
	t.Setenv("SLACKCLI_APP_TOKEN", "xapp-cli")
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-env")
	t.Setenv("SLACK_APP_TOKEN", "xapp-env")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotToken != "xoxb-env" || cfg.AppToken != "xapp-env" {
		t.Fatalf("got %+v, want env tokens to win", cfg)
	}
}

func TestLoadSLACKCLIWhenSlackUnset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	writeCfg(t, path, Config{BotToken: "xoxb-file", AppToken: "xapp-file"})
	t.Setenv("SLACKCLI_CONFIG", path)
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("SLACK_APP_TOKEN", "")
	t.Setenv("SLACKCLI_BOT_TOKEN", "xoxb-cli")
	t.Setenv("SLACKCLI_APP_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotToken != "xoxb-cli" {
		t.Fatalf("bot=%q", cfg.BotToken)
	}
	if cfg.AppToken != "xapp-file" {
		t.Fatalf("app=%q, want file value", cfg.AppToken)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("SLACKCLI_CONFIG", filepath.Join(t.TempDir(), "nope.json"))
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("SLACK_APP_TOKEN", "")
	t.Setenv("SLACKCLI_BOT_TOKEN", "")
	t.Setenv("SLACKCLI_APP_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotToken != "" || cfg.AppToken != "" {
		t.Fatalf("expected empty, got %+v", cfg)
	}
}

func TestNeedBotAndApp(t *testing.T) {
	if err := NeedBot(Config{}); err != ErrMissingBotToken {
		t.Fatalf("empty bot: %v", err)
	}
	if err := NeedBot(Config{BotToken: "xoxp-nope"}); err != ErrBadBotPrefix {
		t.Fatalf("bad bot prefix: %v", err)
	}
	if err := NeedBot(Config{BotToken: "xoxb-ok"}); err != nil {
		t.Fatal(err)
	}
	if err := NeedApp(Config{}); err != ErrMissingAppToken {
		t.Fatalf("empty app: %v", err)
	}
	if err := NeedApp(Config{AppToken: "xoxb-nope"}); err != ErrBadAppPrefix {
		t.Fatalf("bad app prefix: %v", err)
	}
}

func TestSaveMode0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slackcli", "config.json")
	t.Setenv("SLACKCLI_CONFIG", path)
	if err := Save(Config{BotToken: "xoxb-a", AppToken: "xapp-b"}); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", st.Mode().Perm())
	}
}

func writeCfg(t *testing.T, path string, cfg Config) {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}
