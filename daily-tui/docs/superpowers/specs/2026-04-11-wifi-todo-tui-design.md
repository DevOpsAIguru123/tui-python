# daily-tui — WiFi + Todo TUI Design Spec

**Date:** 2026-04-11  
**Status:** Approved

---

## Problem & Goal

Opening a laptop at a coffee shop (e.g. Agora Public) requires manually forgetting yesterday's saved WiFi password and re-entering the new one via System Settings. This is slow and annoying. The goal is a terminal tool that handles this in one keystroke: select the network, enter the new password, done.

The tool lives as a daily-driver TUI dashboard with WiFi management and a persistent todo list.

---

## Tech Stack

- **Language:** Go
- **TUI framework:** Bubble Tea (charmbracelet/bubbletea)
- **Styling:** Lip Gloss (charmbracelet/lipgloss) with Catppuccin Mocha palette
- **WiFi control:** macOS `networksetup` CLI + `airport` for scanning (deprecated on macOS 15+; fallback: `system_profiler SPAirPortDataType`)
- **WiFi interface:** auto-detected at startup via `networksetup -listallhardwareports` (not hardcoded to `en0`)
- **Config:** YAML (`~/.config/daily-tui/config.yaml`)
- **Todo storage:** JSON (`~/.config/daily-tui/todos.json`)

---

## Architecture: Nested Models

The app uses the Bubble Tea nested model pattern. A root `AppModel` owns tab state and delegates messages to the active child model.

```
main.go
├── AppModel            root model, owns active tab index
│   ├── WifiModel       WiFi tab: scanning, connecting, password input
│   └── TodoModel       Todo tab: list, add, complete, delete
│
~/.config/daily-tui/
├── config.yaml         daily-reset network names
└── todos.json          persisted task list
```

### Message flow
1. `AppModel.Update()` receives all keypresses
2. Tab-switch keys (`tab`, `1`, `2`) are handled at root level
3. All other keys are forwarded to the active child's `Update()`
4. Children return `tea.Cmd` for async operations (network scan, shell commands)
5. Results come back as custom `tea.Msg` types handled in each child's `Update()`

---

## Config Format

Location: `~/.config/daily-tui/config.yaml`

```yaml
wifi:
  daily_reset_networks:
    - "Agora Public"
    - "CoffeeShop_Guest"
```

Networks listed under `daily_reset_networks` trigger the forget + reprompt flow automatically when selected. All other networks use the standard connect flow.

---

## WiFi Tab

### Layout

```
┌─ WiFi ──────────────────────────────────────────────┐
│  ● Connected: Agora Public                          │
│                                                     │
│  Nearby Networks                                    │
│  ▸ Agora Public        ████████  -45 dBm  [daily]  │
│    CoffeeShop_2G       ██████    -62 dBm            │
│    iPhone Hotspot      ████      -71 dBm            │
│                                                     │
│  ↑↓ navigate  •  enter to connect  •  r refresh    │
└─────────────────────────────────────────────────────┘
```

`[daily]` badge appears next to networks in the config's `daily_reset_networks` list.

### Daily-reset network flow (e.g. Agora Public)
1. User highlights network and presses `enter`
2. Password input field appears inline (bottom of panel)
3. On confirm (`enter`):
   - Run: `networksetup -removepreferredwirelessnetwork <wifi_iface> "Agora Public"`
   - Run: `networksetup -setairportnetwork <wifi_iface> "Agora Public" "<password>"`
   - `<wifi_iface>` is resolved at startup (e.g. `en0`, `en1`) and cached
4. Status line updates: `Connecting...` → `● Connected: Agora Public` or `✗ Wrong password`

### Normal network flow
1. User presses `enter` on a non-daily-reset network
2. If already a known/saved network: connect directly, no password prompt
3. If new network: password input appears inline

### Network scanning
- Runs `airport -s` on launch and every 30 seconds in background
- If `airport` is unavailable (macOS 15+), falls back to `system_profiler SPAirPortDataType` (slower, ~3s)
- `r` key triggers immediate rescan
- Signal strength rendered as block bar (1–4 blocks based on dBm)
- Saved networks detected via `networksetup -listpreferredwirelessnetworks <wifi_iface>` to determine if a password prompt is needed

### Keybindings
| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate network list |
| `enter` | Connect / confirm password |
| `esc` | Cancel password input |
| `r` | Refresh network list |

---

## Todo Tab

### Layout

```
┌─ Todo ──────────────────────────────────────────────┐
│  Today's Tasks                                      │
│                                                     │
│  ▸ ☐  Morning standup                              │
│    ☐  Review PRs                                    │
│    ☑  Buy coffee                                    │
│    ☑  Open laptop                                   │
│                                                     │
│  [ Add a task...                                  ] │
│                                                     │
│  ↑↓ navigate  •  space complete  •  d delete  •  a add │
└─────────────────────────────────────────────────────┘
```

### Persistence
- Tasks saved to `~/.config/daily-tui/todos.json` on every mutation (add, complete, delete)
- No explicit save step — changes are immediate
- Tasks carry over between sessions indefinitely

### Keybindings
| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate task list |
| `space` | Toggle complete / incomplete |
| `d` | Delete selected task |
| `a` | Focus add-task input |
| `enter` | Confirm new task |
| `esc` | Cancel input |

---

## Global Navigation

| Key | Action |
|-----|--------|
| `tab` / `1` / `2` | Switch tabs |
| `q` / `ctrl+c` | Quit |
| `?` | Toggle help overlay |

Active tab: highlighted in Catppuccin Mauve (`#cba6f7`)  
Inactive tabs: dimmed (`#585b70`)

---

## Catppuccin Mocha Palette (key colors)

| Role | Color | Hex |
|------|-------|-----|
| Background | Base | `#1e1e2e` |
| Surface | Surface0 | `#313244` |
| Active tab / accent | Mauve | `#cba6f7` |
| Connected (green) | Green | `#a6e3a1` |
| Error (red) | Red | `#f38ba8` |
| Text | Text | `#cdd6f4` |
| Dimmed / inactive | Overlay0 | `#6c7086` |
| Daily badge | Peach | `#fab387` |

---

## File Structure

```
daily-tui/
├── main.go
├── go.mod
├── go.sum
├── internal/
│   ├── app/
│   │   └── model.go        AppModel: root model, tab switching
│   ├── wifi/
│   │   ├── model.go        WifiModel: state, Update, View
│   │   ├── commands.go     tea.Cmd wrappers for networksetup/airport
│   │   └── scanner.go      airport -s parsing
│   ├── todo/
│   │   ├── model.go        TodoModel: state, Update, View
│   │   └── store.go        JSON read/write for todos.json
│   ├── config/
│   │   └── config.go       YAML config loading
│   └── theme/
│       └── theme.go        Lip Gloss styles, Catppuccin Mocha palette
└── docs/
    └── superpowers/specs/
        └── 2026-04-11-wifi-todo-tui-design.md
```

---

## Verification

1. `go run .` launches the TUI
2. WiFi tab shows nearby networks with signal bars
3. Selecting `Agora Public` shows password prompt immediately (no confirmation dialog)
4. Entering password: app forgets old network, reconnects with new password, status updates
5. Selecting a normal network connects without prompting for password
6. Todo tab: add task, complete task, delete task — all persist after quit and reopen
7. Tab key switches between WiFi and Todo tabs
8. `q` quits cleanly
