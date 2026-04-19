"""Root Textual app: sidebar with tabs, main pane with Todo/Claude content."""

from __future__ import annotations

import getpass
import os
from datetime import datetime
from pathlib import Path

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, Vertical
from textual.reactive import reactive
from textual.widgets import Static

from . import __version__, theme
from .claude import ClaudeView
from .todo import TodoCountChanged, TodoStore, TodoView


TUX_ART = r"""
  ▄▀▀▀▀▄
 █ ●  ● █
 █  ▄▄  █
 ▀█▄▄▄▄█▀
  █    █
  ▀▀  ▀▀
"""


def config_dir() -> Path:
    return Path(os.path.expanduser("~/.config/daily-tui-py"))


class Sidebar(Vertical):
    DEFAULT_CSS = f"""
    Sidebar {{
        width: 22;
        padding: 1 2;
        background: {theme.MANTLE};
        border-right: solid {theme.SURFACE1};
    }}
    Sidebar .brand {{
        color: {theme.MAUVE};
        text-style: bold;
    }}
    Sidebar .brand-sub {{
        color: {theme.OVERLAY};
        padding-bottom: 1;
    }}
    Sidebar .tux {{
        color: {theme.TEXT};
        padding-bottom: 1;
    }}
    Sidebar .section {{
        color: {theme.OVERLAY};
        padding-top: 1;
    }}
    Sidebar .tabs {{
        padding-top: 0;
    }}
    Sidebar .host {{
        color: {theme.OVERLAY};
    }}
    """

    active_tab: reactive[str] = reactive("todo")
    todo_count: reactive[int] = reactive(0)

    def compose(self) -> ComposeResult:
        yield Static(f"[bold {theme.MAUVE}]daily-tui-py[/]", classes="brand")
        yield Static(f"[{theme.OVERLAY}]v{__version__}[/]", classes="brand-sub")
        yield Static(f"[{theme.YELLOW}]{TUX_ART}[/]", classes="tux")
        yield Static(f"[{theme.OVERLAY}]TABS[/]", classes="section")
        yield Static("", id="tabs", classes="tabs")
        yield Static(f"[{theme.OVERLAY}]HOST[/]", classes="section")
        yield Static(self._host_block(), classes="host")
        yield Static(f"[{theme.OVERLAY}]KEYS[/]", classes="section")
        yield Static(self._keys_block(), classes="host")

    def on_mount(self) -> None:
        self._refresh_tabs()

    def watch_active_tab(self, _old: str, _new: str) -> None:
        self._refresh_tabs()

    def watch_todo_count(self, _old: int, _new: int) -> None:
        self._refresh_tabs()

    def _refresh_tabs(self) -> None:
        lines = []
        for key, glyph, label, pill in [
            ("todo", "●", "Todo", f"{self.todo_count}"),
            ("claude", "✦", "Claude", ""),
        ]:
            active = key == self.active_tab
            glyph_style = theme.MAUVE if active else theme.OVERLAY
            name_style = (
                f"bold {theme.MAUVE}" if active else theme.SUBTEXT0
            )
            row = (
                f"[{glyph_style}]{glyph}[/]  "
                f"[{name_style}]{label}[/]"
            )
            if pill:
                row += f"  [{theme.OVERLAY} on {theme.SURFACE}] {pill} [/]"
            lines.append(row)
        self.query_one("#tabs", Static).update("\n".join(lines))

    def _host_block(self) -> str:
        try:
            user = getpass.getuser()
        except Exception:
            user = os.environ.get("USER", "user")
        return (
            f"[{theme.OVERLAY}]$ whoami[/]\n"
            f"[{theme.GREEN}]{user}[/]"
        )

    def _keys_block(self) -> str:
        return (
            f"[{theme.SUBTEXT0}]tab[/] [{theme.OVERLAY}]switch[/]\n"
            f"[{theme.SUBTEXT0}]1/2[/] [{theme.OVERLAY}]tabs[/]\n"
            f"[{theme.SUBTEXT0}]q[/]   [{theme.OVERLAY}]quit[/]"
        )


class MainPane(Vertical):
    """Right-hand column: breadcrumb header + active tab content."""

    DEFAULT_CSS = f"""
    MainPane {{
        padding: 1 2;
    }}
    MainPane #breadcrumb {{
        color: {theme.TEXT};
        padding-bottom: 1;
    }}
    MainPane #tab-container {{
        height: 1fr;
    }}
    """

    def __init__(self, todo: TodoView, claude: ClaudeView) -> None:
        super().__init__()
        self.todo = todo
        self.claude = claude

    def compose(self) -> ComposeResult:
        yield Static("", id="breadcrumb")
        yield Vertical(self.todo, self.claude, id="tab-container")


class DailyTuiApp(App):
    """Top-level app: owns the store, views, and tab switching."""

    CSS = f"""
    Screen {{
        background: {theme.BASE};
        color: {theme.TEXT};
    }}
    #root {{
        height: 1fr;
    }}
    """

    BINDINGS = [
        Binding("tab", "next_tab", "switch", show=False, priority=True),
        Binding("1", "switch_tab('todo')", "todo", show=False),
        Binding("2", "switch_tab('claude')", "claude", show=False),
        Binding("q", "quit", "quit", show=False),
        Binding("ctrl+c", "quit", "quit", show=False),
    ]

    active_tab: reactive[str] = reactive("todo")

    def __init__(self) -> None:
        super().__init__()
        cfg = config_dir()
        cfg.mkdir(parents=True, exist_ok=True)
        self.store = TodoStore(cfg / "todos.json")
        self.config_path = cfg / "config.yaml"
        self.todo_view = TodoView(self.store)
        self.claude_view = ClaudeView(self.config_path)
        self.sidebar = Sidebar()
        self.main = MainPane(self.todo_view, self.claude_view)

    def compose(self) -> ComposeResult:
        yield Horizontal(self.sidebar, self.main, id="root")

    def on_mount(self) -> None:
        self._apply_active_tab()
        self._refresh_breadcrumb()
        self.set_interval(30.0, self._refresh_breadcrumb)

    def _apply_active_tab(self) -> None:
        self.sidebar.active_tab = self.active_tab
        self.todo_view.display = self.active_tab == "todo"
        self.claude_view.display = self.active_tab == "claude"
        target = self.todo_view if self.active_tab == "todo" else self.claude_view
        try:
            target.focus()
        except Exception:
            pass
        self._refresh_breadcrumb()

    def watch_active_tab(self, _old: str, _new: str) -> None:
        self._apply_active_tab()

    def action_next_tab(self) -> None:
        self.active_tab = "claude" if self.active_tab == "todo" else "todo"

    def action_switch_tab(self, name: str) -> None:
        if name in ("todo", "claude"):
            self.active_tab = name

    def _refresh_breadcrumb(self) -> None:
        if self.active_tab == "todo":
            label = "Todo"
            sub = self.todo_view.title()
        else:
            label = "Claude"
            sub = self.claude_view.title()
        clock = datetime.now().strftime("%I:%M %p").lstrip("0")
        left = (
            f"[{theme.MAUVE}]›[/] "
            f"[bold {theme.TEXT}]{label}[/] "
            f"[{theme.SURFACE1}]·[/] "
            f"[{theme.SUBTEXT0}]{sub}[/]"
        )
        right = f"[{theme.OVERLAY}]{clock}[/]"
        self.main.query_one("#breadcrumb", Static).update(f"{left}    {right}")

    def on_todo_count_changed(self, event: TodoCountChanged) -> None:
        self.sidebar.todo_count = event.pending
        self._refresh_breadcrumb()
