# slackcli

Post to Slack as a bot and listen for replies in a long-running terminal process.

This is not Slack’s official app-dev CLI (`slack`). The binary is `slackcli` so the two can coexist.

## Install

```bash
go install github.com/timjonez/slack-cli/cmd/slackcli@latest
# or from a checkout:
make install
```

Requires Go 1.25+.

## Slack app setup

1. Create an app from [manifest.yaml](manifest.yaml) at [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From a manifest**.
2. Under **Basic Information → App-Level Tokens**, generate a token with the `connections:write` scope. It starts with `xapp-`.
3. **Install App** to your workspace. Copy the **Bot User OAuth Token** (`xoxb-`).
4. Invite the bot to any **private** channel you want it to see (`/invite @slackcli`). Public channels can be joined with `slackcli join #name`.
5. If you already created the app before Messages was enabled: **App Home → Show Tabs → Messages** — turn the tab on and allow users to send messages (uncheck read-only). Without that, Slack shows “Sending messages to this app has been turned off.”

```bash
slackcli auth set --bot-token xoxb-... --app-token xapp-...
slackcli whoami
```

Tokens can also live in the environment. Resolution order (first non-empty wins per field):

1. `SLACK_BOT_TOKEN` / `SLACK_APP_TOKEN`
2. `SLACKCLI_BOT_TOKEN` / `SLACKCLI_APP_TOKEN`
3. `$XDG_CONFIG_HOME/slackcli/config.json` (or `~/.config/slackcli/config.json`)

Override the file path with `SLACKCLI_CONFIG`. The file is written mode `0600`. Tokens are never printed.

`send`, `whoami`, `channels`, and `join` need the bot token. `listen` needs both.

## Commands

```bash
slackcli send #eng "deploy complete"
slackcli send #eng --thread 1710000000.000100 "follow-up"
echo "from pipe" | slackcli send #eng

slackcli listen                 # every conversation the bot is in
slackcli listen #eng
slackcli listen #eng --mentions
slackcli listen #eng --thread 1710000000.000100

slackcli whoami
slackcli channels
slackcli join #eng
slackcli auth path
slackcli version
```

`--json` prints machine-readable JSON on stdout (JSONL for `listen`). Errors go to stderr as `{"error":"...","code":"..."}`. Status from `listen` (`connecting`, `connected`) is always on stderr so the message stream stays pipeable.

`listen` is a blocking Socket Mode client. Background it yourself:

```bash
# another pane / tmux
slackcli listen #eng

# or a user systemd unit
# ~/.config/systemd/user/slackcli-listen.service
# ExecStart=/home/you/go/bin/slackcli listen
```

## Channel names

`CHANNEL` is `#eng`, `eng`, or a Slack id (`C…` / `D…` / `G…`). Names are resolved to ids before posting. The bot must be **in** a channel to receive its events; `send` will try `conversations.join` once if Slack returns `not_in_channel` on a public channel.

## Development

```bash
make test
make build   # ./bin/slackcli
```
