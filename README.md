# daily-tui-py

A Python port of the `daily-tui` terminal dashboard — sidebar layout, Catppuccin
Mocha palette, and two tabs: **Todo** and **Claude commands**. Built with
[Textual](https://textual.textualize.io/).

This is a focused subset of the Go original (`../daily-tui`): no WiFi, no
Calendar, no Portfolio — just task tracking and a launcher for `claude -p`
commands.

---

## Layout

```
┌────────────┬──────────────────────────────────────────────────┐
│ daily-tui  │  › Todo · 2 pending · 1 done          05:30 PM   │
│ v0.1.0     │                                                   │
│            │  Today's Tasks  2 open · 1 done                   │
│ [tux]      │  ── TASKS ─────────────────────────────────────   │
│            │  ▸  ☐  Morning standup                     #01    │
│ TABS       │     ☐  Review PRs                          #02    │
│ ● Todo  2  │     ☑  Buy coffee                          #03    │
│ ✦ Claude   │                                                   │
│            │  + Press a to add a task…                         │
│ HOST       │                                                   │
│ $ whoami   │                                                   │
│ zach       │                                                   │
│            │  [↑↓] navigate  [space] toggle  [a] add …         │
│ KEYS       │                                                   │
│ tab switch │                                                   │
│ 1/2 tabs   │                                                   │
│ q   quit   │                                                   │
└────────────┴──────────────────────────────────────────────────┘
```

---

## Install & run (uv)

```bash
cd daily-tui-py
uv sync                 # create .venv, install deps (textual, pyyaml)
uv run daily-tui-py     # launch the TUI
```

One-off run without syncing first:

```bash
uv run --project daily-tui-py daily-tui-py
```

Install globally as a tool:

```bash
uv tool install --from . daily-tui-py
daily-tui-py
```

### Requirements

- Python ≥ 3.9
- `uv` (install from <https://docs.astral.sh/uv/>)
- `claude` CLI on `$PATH` (only needed for the Claude tab to actually run
  commands; everything else works without it)

---

## Config

Files live under `~/.config/daily-tui-py/`:

| File            | Purpose                              |
|-----------------|--------------------------------------|
| `todos.json`    | Persistent task list (auto-created)  |
| `config.yaml`   | Optional; user-defined Claude entries|

Optional `config.yaml` shape (same schema as the Go version's `claude_code`
section):

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

Each entry becomes a row in the Claude tab. Pressing Enter runs
`claude -p "<prompt>" [extra_args...]`.

---

## Todo tab

| Key      | Action                    |
|----------|---------------------------|
| `↑` `↓`  | Navigate task list        |
| `space`  | Toggle complete           |
| `a`      | Add a new task            |
| `e`      | Edit selected task        |
| `d`      | Delete selected task      |
| `enter`  | Confirm add / edit        |
| `esc`    | Cancel input              |

Tasks are split into pending (top) then completed (bottom, struck-through).
Storage is a plain JSON file — you can hand-edit it if needed.

## Claude tab

Commands are discovered in this order (stable across runs):

1. Built-ins (`hello`, `network-monitor`)
2. Entries from `~/.config/daily-tui-py/config.yaml`
3. Markdown files under `~/.claude/commands/*.md` (tagged `user`)
4. Markdown files under `$PWD/.claude/commands/*.md` (tagged `project`)

| Key      | Action                                |
|----------|---------------------------------------|
| `↑` `↓`  | Navigate command list                 |
| `enter`  | Run the selected command              |
| `esc`    | Clear the focused command's last run  |

Each command tracks its own run state, so multiple commands can be in flight
at once. A 5-minute timeout bounds any single invocation.

## Global

| Key        | Action       |
|------------|--------------|
| `tab`      | Switch tabs  |
| `1` / `2`  | Todo / Claude|
| `q`        | Quit         |
| `ctrl+c`   | Quit         |

---

## Project layout

```
daily-tui-py/
├── pyproject.toml
├── README.md
└── daily_tui/
    ├── __init__.py
    ├── __main__.py       # CLI entrypoint
    ├── app.py            # Textual App: sidebar + tab switching
    ├── theme.py          # Catppuccin Mocha palette
    ├── todo.py           # Todo widget + JSON store
    └── claude.py         # Claude commands widget + async runner
```

---

## Relationship to the Go version

This Python port intentionally covers only Todo + Claude. The Go app under
`../daily-tui` remains the full-featured version (WiFi daily-reset flow,
macOS Calendar, Portfolio). They share no state — different config
directories (`~/.config/daily-tui/` vs `~/.config/daily-tui-py/`) so you can
run both side by side.
