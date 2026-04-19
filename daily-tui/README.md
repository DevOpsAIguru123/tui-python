# daily-tui

A terminal dashboard for macOS that solves one specific annoyance: coffee shops that reset their WiFi password every day. Select the network, type the new password — it automatically forgets the old one and reconnects. No fumbling through System Settings.

Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea), styled with the [Catppuccin Mocha](https://catppuccin.com) palette.

---

## Features

### WiFi Tab
- Lists all saved (preferred) WiFi networks
- **Daily-reset flow** — networks listed in config always prompt for a fresh password, auto-forget the old entry, then reconnect
- **Normal saved networks** — connect in one keystroke, no password prompt
- `r` to refresh the network list
- Works on macOS 15+ (Sequoia) — uses `networksetup` instead of the deprecated `airport -s` which redacts SSIDs on newer macOS

### Todo Tab
- Persistent task list saved to `~/.config/daily-tui/todos.json`
- Add, complete, and delete tasks
- Tasks carry over between sessions

### Calendar Tab
- Read-only view of upcoming events from macOS Calendar.app (next 7 days)
- Events fetched via `osascript` — no OAuth, no extra setup
- First run may prompt "Terminal wants to access Calendar" in System Settings → Privacy → Automation

---

## Requirements

- macOS (uses `networksetup` CLI — macOS only)
- Go 1.21+

---

## Install

### go install (recommended)

Install directly to your `$GOPATH/bin` (or `$GOBIN`) — runs from any directory:

```bash
go install github.com/DevOpsAIguru123/productivity-tools/daily-tui@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### From source with make

```bash
git clone https://github.com/DevOpsAIguru123/productivity-tools.git
cd productivity-tools/daily-tui
make install          # builds and copies to /usr/local/bin
```

To install elsewhere:

```bash
make install PREFIX=$HOME/.local
```

To uninstall:

```bash
make uninstall
```

### From source (manual)

```bash
git clone https://github.com/DevOpsAIguru123/productivity-tools.git
cd productivity-tools/daily-tui
go build -o daily-tui .
sudo cp daily-tui /usr/local/bin/
```

---

## Configuration

Config file: `~/.config/daily-tui/config.yaml`

Created automatically on first run. Edit it to add your daily-reset networks:

```yaml
wifi:
  daily_reset_networks:
    - "Agora Public"
    - "CoffeeShop_Guest"
```

Any network listed here will **always** prompt for a fresh password when you press Enter — even if it's already saved. The old entry is automatically removed before reconnecting. All other networks connect silently using the saved credential.

You can also add custom launcher entries for the **Claude tab** — these are merged with the built-ins and any `~/.claude/commands/*.md` files:

```yaml
claude_code:
  commands:
    - name: deploy
      prompt: "run the deploy pipeline and summarise output"
    - name: security-scan
      prompt: "run /security-review"
      extra_args:
        - "--dangerously-skip-permissions"
```

Each entry becomes a row in the Claude tab; pressing Enter runs `claude -p "<prompt>" [extra_args...]`. Entries with a blank `name` or `prompt` are skipped. To invoke a saved Claude Code slash command server-side, reference it from the `prompt` (e.g. `"run /security-review"`) rather than re-stating the template.

---

## Usage

```bash
daily-tui            # launch the TUI
daily-tui --version  # print version
daily-tui --help     # show help
```

### WiFi tab

```
○ Not connected

Saved Networks
▸ Agora Public   [daily]
  HomeNetwork
  iPhone Hotspot

↑↓ navigate • enter connect • r refresh • esc cancel
```

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate network list |
| `enter` | Connect (prompts for password on daily-reset networks) |
| `esc` | Cancel password input |
| `r` | Refresh network list |

**Daily-reset network flow:**
1. Navigate to the network and press `enter`
2. Password field appears inline at the bottom
3. Type the new password and press `enter`
4. App forgets yesterday's saved password and reconnects with the new one
5. Status updates to `● Connected: <network>`

### Todo tab

```
Today's Tasks

▸ ☐  Morning standup
  ☐  Review PRs
  ☑  Buy coffee

↑↓ navigate • space complete • d delete • a add
```

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate task list |
| `space` | Toggle complete / incomplete |
| `d` | Delete selected task |
| `a` | Open add-task input |
| `enter` | Confirm new task |
| `esc` | Cancel input |

### Calendar tab

```
Upcoming — next 7 days

── Fri, Apr 17 ──────────────────────
▸ 9:00am–9:15am   Standup  Work
  2:00pm–3:00pm   Dentist  @ Downtown  Personal

↑↓ nav • r refresh
```

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate event list |
| `r` | Refresh from Calendar.app |

### Global

| Key | Action |
|-----|--------|
| `tab` | Cycle through tabs |
| `1` | Go to WiFi tab |
| `2` | Go to Todo tab |
| `3` | Go to Calendar tab |
| `q` / `ctrl+c` | Quit |

---

## How it works

### WiFi on macOS 15+

macOS Sequoia redacts SSIDs in `airport -s` and `system_profiler SPAirPortDataType` output for privacy. This tool uses `networksetup -listpreferredwirelessnetworks` instead, which returns real network names.

The daily-reset flow shells out to:

```bash
# Remove old saved password
networksetup -removepreferredwirelessnetwork en0 "Agora Public"

# Connect with new password
networksetup -setairportnetwork en0 "Agora Public" "<password>"
```

The WiFi interface (`en0`, `en1`, etc.) is auto-detected at startup via `networksetup -listallhardwareports` — nothing is hardcoded.

### Architecture

```
main.go
└── AppModel              root model, owns tab state
    ├── WifiModel         WiFi tab: state machine, async shell commands
    ├── TodoModel         Todo tab: task list, text input
    └── CalendarModel     Calendar tab: read-only agenda via osascript

~/.config/daily-tui/
├── config.yaml           daily-reset network names
└── todos.json            persisted task list
```

The app follows the [Bubble Tea nested model pattern](https://github.com/charmbracelet/bubbletea/tree/master/examples). `AppModel` delegates all messages to the active child model. Tab-switching keys (`tab`, `1`, `2`) are only intercepted at the root level when no input field is active — so typing a password or a task name is never hijacked.

WiFi operations are fully async: each shell command is wrapped in a `tea.Cmd` that runs in a goroutine and returns a typed message when done.

---

## Project structure

```
daily-tui/
├── main.go
├── go.mod
├── internal/
│   ├── app/
│   │   ├── model.go          AppModel: root, tab switching
│   │   └── model_test.go
│   ├── wifi/
│   │   ├── model.go          WifiModel: state machine, Update, View
│   │   ├── commands.go       tea.Cmd wrappers for networksetup
│   │   ├── scanner.go        preferred-network parser, interface detection
│   │   └── *_test.go
│   ├── todo/
│   │   ├── model.go          TodoModel: state, Update, View
│   │   ├── store.go          JSON persistence
│   │   └── *_test.go
│   ├── calendar/
│   │   ├── model.go          CalendarModel: agenda view
│   │   ├── source.go         osascript fetch + parser
│   │   ├── commands.go       tea.Cmd wrappers
│   │   └── *_test.go
│   ├── config/
│   │   ├── config.go         YAML config loading
│   │   └── config_test.go
│   └── theme/
│       └── theme.go          Catppuccin Mocha lip gloss styles
└── docs/
    └── superpowers/specs/    design spec
```

---

## Development

```bash
# Run tests
make test

# Build
make build          # output: bin/daily-tui

# Build + install to /usr/local/bin
make install

# Clean build artifacts
make clean
```

---

## Roadmap

- [ ] Internet connectivity check after connect — show `● Connected` vs `● Connected (no internet)` by pinging a known host ([#1](https://github.com/DevOpsAIguru123/productivity-tools/issues/1))
- [ ] Mid-session password reset — re-enter password for the currently connected daily-reset network without navigating away ([#1](https://github.com/DevOpsAIguru123/productivity-tools/issues/1))

---

## Dependencies

| Package | Role |
|---------|------|
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm-style) |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) | Text input component |
| [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | Config parsing |
