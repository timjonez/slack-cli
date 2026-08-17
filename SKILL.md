---
name: slackcli
description: >
  Use when sending Slack messages as a bot or listening for channel replies
  with the slackcli CLI. Triggers: slackcli, post to Slack, send as bot,
  listen to a Slack channel, Slack Socket Mode.
---

# slackcli

Post and listen as a Slack **bot**. Binary: `slackcli` (not Slack’s official `slack` CLI).

**Tokens:** `SLACK_BOT_TOKEN` (`xoxb-`) / `SLACK_APP_TOKEN` (`xapp-`), then `SLACKCLI_*`, then `$XDG_CONFIG_HOME/slackcli/config.json`. Override path: `SLACKCLI_CONFIG`. Never print tokens.

`send` / `whoami` / `channels` / `join` need the bot token. `listen` needs both.

**Channel:** `#eng`, `eng`, or `C…`/`D…`/`G…` id. Bot must be **in** the channel to receive events. Private channels need a human `/invite`.

## Rules

- Prefer `slackcli` over ad-hoc Slack API calls.
- **Always pass `--json`** when scripting or parsing. Human output is for people. `listen --json` is JSONL (one event per line).
- Status (`connecting`, `connected`) is on **stderr**. Message stream is on **stdout**.
- `listen` is blocking. Background it (tmux/pane/`systemd --user`); do not treat it as a one-shot.
- On `--json` failure, read stderr: `{"error":"...","code":"not_found|invalid|<slack_error>|error"}`.
- Do not invent channel names — `channels` or ask. Confirm before posting to a busy/public channel if the user was ambiguous.

## Commands

```bash
slackcli --json whoami
slackcli --json auth path
slackcli auth set --bot-token xoxb-... --app-token xapp-...

slackcli --json send CHANNEL [TEXT...] [--thread TS]
echo "body" | slackcli --json send CHANNEL          # or TEXT=-

slackcli --json listen [CHANNEL...] [--mentions] [--thread TS]
slackcli --json channels
slackcli --json join CHANNEL
slackcli --json version
```

## Typical flow

```bash
slackcli --json whoami
slackcli --json channels
slackcli --json send #eng "deploy complete"
# other pane:
slackcli --json listen #eng
```
