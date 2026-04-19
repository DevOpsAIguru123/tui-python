"""Tmux tab: list sessions, create/kill, attach with suspend/resume."""

from __future__ import annotations

import subprocess
from dataclasses import dataclass
from typing import List, Optional

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Vertical
from textual.message import Message
from textual.widgets import Input, Static

from . import theme


@dataclass
class Session:
    name: str
    windows: int
    attached: bool
    created: str = ""


class TmuxCountChanged(Message):
    def __init__(self, count: int) -> None:
        self.count = count
        super().__init__()


class TmuxAttachRequested(Message):
    def __init__(self, name: str) -> None:
        self.name = name
        super().__init__()


def _run(argv: List[str]) -> subprocess.CompletedProcess:
    return subprocess.run(
        argv,
        capture_output=True,
        text=True,
        check=False,
    )


def list_sessions() -> List[Session]:
    fmt = "#{session_name}|#{session_windows}|#{session_attached}|#{session_created}"
    try:
        res = _run(["tmux", "list-sessions", "-F", fmt])
    except FileNotFoundError:
        return []
    if res.returncode != 0:
        # "no server running" is not an error in this UI — it just means 0 sessions.
        return []
    sessions: List[Session] = []
    for line in res.stdout.splitlines():
        parts = line.split("|")
        if len(parts) < 3:
            continue
        name, windows, attached = parts[0], parts[1], parts[2]
        created = parts[3] if len(parts) > 3 else ""
        try:
            sessions.append(
                Session(
                    name=name,
                    windows=int(windows or "0"),
                    attached=attached == "1",
                    created=created,
                )
            )
        except ValueError:
            continue
    return sessions


def create_session(name: str) -> Optional[str]:
    """Create a detached session. Returns None on success, error string otherwise."""
    if not name.strip():
        return "empty session name"
    try:
        res = _run(["tmux", "new-session", "-d", "-s", name])
    except FileNotFoundError:
        return "`tmux` binary not found on PATH"
    if res.returncode != 0:
        return (res.stderr or res.stdout or "failed to create session").strip()
    return None


def kill_session(name: str) -> Optional[str]:
    try:
        res = _run(["tmux", "kill-session", "-t", name])
    except FileNotFoundError:
        return "`tmux` binary not found on PATH"
    if res.returncode != 0:
        return (res.stderr or res.stdout or "failed to kill session").strip()
    return None


class TmuxView(Vertical):
    """Session list + inline 'new session' input."""

    DEFAULT_CSS = f"""
    TmuxView {{
        padding: 0 0;
    }}
    TmuxView Input {{
        border: tall {theme.SURFACE1};
        background: {theme.BASE};
        color: {theme.TEXT};
        margin-top: 1;
    }}
    TmuxView Input.-hidden {{
        display: none;
    }}
    TmuxView .help-bar {{
        color: {theme.OVERLAY};
        padding-top: 1;
    }}
    """

    BINDINGS = [
        Binding("up", "cursor_up", "up", show=False),
        Binding("down", "cursor_down", "down", show=False),
        Binding("enter", "attach", "attach"),
        Binding("n", "new", "new"),
        Binding("k", "kill", "kill"),
        Binding("r", "refresh", "refresh"),
        Binding("escape", "cancel", "cancel", show=False),
    ]

    can_focus = True

    def __init__(self) -> None:
        super().__init__()
        self.sessions: List[Session] = []
        self.cursor = 0
        self.mode: Optional[str] = None  # None | "new"
        self.status: str = ""
        self.status_is_error: bool = False

    def compose(self) -> ComposeResult:
        yield Static("", id="tmux-body")
        inp = Input(placeholder="New session name…", id="tmux-input")
        inp.add_class("-hidden")
        inp.can_focus = False
        yield inp
        yield Static("", id="tmux-help", classes="help-bar")

    def on_mount(self) -> None:
        self.refresh_list()

    def is_inputting(self) -> bool:
        return self.mode is not None

    def title(self) -> str:
        n = len(self.sessions)
        if n == 0:
            return "no sessions"
        return f"{n} session{'s' if n != 1 else ''}"

    def count(self) -> int:
        return len(self.sessions)

    # ---- data ----

    def refresh_list(self) -> None:
        self.sessions = list_sessions()
        if self.cursor >= len(self.sessions):
            self.cursor = max(0, len(self.sessions) - 1)
        self._refresh_ui()
        self.post_message(TmuxCountChanged(len(self.sessions)))

    # ---- rendering ----

    def _refresh_ui(self) -> None:
        lines: list[str] = []
        n = len(self.sessions)

        # Header — matches the mockup style: "Tmux Sessions   N active"
        if n == 0:
            header = (
                f"[{theme.TEXT}]Tmux Sessions[/]   "
                f"[{theme.OVERLAY}]0 active[/]"
            )
        else:
            attached = sum(1 for s in self.sessions if s.attached)
            header = (
                f"[{theme.TEXT}]Tmux Sessions[/]   "
                f"[{theme.PEACH}]{n} active[/]   "
                f"[{theme.OVERLAY}]{attached} attached[/]"
            )
        lines.append(header)
        lines.append(f"[{theme.OVERLAY}]── SESSIONS {'─' * 56}[/]")

        if n == 0:
            lines.append(
                f"[{theme.OVERLAY}]  No tmux sessions — press 'n' to create one[/]"
            )
        else:
            for i, s in enumerate(self.sessions):
                selected = (i == self.cursor) and not self.is_inputting()
                lines.append(self._render_row(s, selected))

        if self.status:
            lines.append("")
            color = theme.RED if self.status_is_error else theme.GREEN
            lines.append(f"[{color}]  {self.status}[/]")

        if self.is_inputting():
            lines.append("")
            lines.append(f"[{theme.OVERLAY}]  Enter a session name, press enter to create + attach[/]")

        self.query_one("#tmux-body", Static).update("\n".join(lines))
        self.query_one("#tmux-help", Static).update(self._help_bar())

    def _render_row(self, s: Session, selected: bool) -> str:
        caret = f"[{theme.MAUVE}]▸[/]" if selected else " "
        dot = (
            f"[{theme.GREEN}]●[/]"
            if s.attached
            else f"[{theme.OVERLAY}]○[/]"
        )
        name_style = f"bold {theme.TEXT}" if selected else theme.TEXT
        name = f"[{name_style}]{s.name}[/]"
        meta = f"[{theme.OVERLAY}]{s.windows} window{'s' if s.windows != 1 else ''}[/]"
        tag_text = "attached" if s.attached else "detached"
        tag_color = theme.GREEN if s.attached else theme.SUBTEXT0
        tag = f"[{tag_color} on {theme.SURFACE}] {tag_text} [/]"
        return f"{caret}  {dot}  {name}  {meta}  {tag}"

    def _help_bar(self) -> str:
        if self.mode == "new":
            hints = [("enter", "create + attach"), ("esc", "cancel")]
        else:
            hints = [
                ("↑↓", "navigate"),
                ("enter", "attach"),
                ("n", "new"),
                ("k", "kill"),
                ("r", "refresh"),
            ]
        parts = []
        for k, label in hints:
            parts.append(
                f"[{theme.TEXT} on {theme.SURFACE}] {k} [/] "
                f"[{theme.OVERLAY}]{label}[/]"
            )
        bar = "  ".join(parts)
        if self.mode is None and self.sessions:
            bar += f"\n  [{theme.OVERLAY}]tip: inside tmux, press Ctrl-B then D to detach back to this tab[/]"
        return bar

    # ---- actions ----

    def action_cursor_up(self) -> None:
        if self.is_inputting():
            return
        if self.cursor > 0:
            self.cursor -= 1
            self._refresh_ui()

    def action_cursor_down(self) -> None:
        if self.is_inputting():
            return
        if self.cursor < len(self.sessions) - 1:
            self.cursor += 1
            self._refresh_ui()

    def action_refresh(self) -> None:
        if self.is_inputting():
            return
        self.status = ""
        self.refresh_list()

    def action_new(self) -> None:
        if self.is_inputting():
            return
        self.mode = "new"
        self.status = ""
        inp = self.query_one("#tmux-input", Input)
        inp.value = ""
        inp.placeholder = "New session name…"
        inp.remove_class("-hidden")
        inp.can_focus = True
        inp.focus()
        self._refresh_ui()

    def action_kill(self) -> None:
        if self.is_inputting():
            return
        if not self.sessions:
            return
        target = self.sessions[self.cursor].name
        err = kill_session(target)
        if err:
            self.status = f"kill failed: {err}"
            self.status_is_error = True
        else:
            self.status = f"killed session '{target}'"
            self.status_is_error = False
        self.refresh_list()

    def action_attach(self) -> None:
        if self.is_inputting():
            return
        if not self.sessions:
            return
        name = self.sessions[self.cursor].name
        self.post_message(TmuxAttachRequested(name))

    def action_cancel(self) -> None:
        if not self.is_inputting():
            return
        self._close_input()

    def _close_input(self) -> None:
        self.mode = None
        inp = self.query_one("#tmux-input", Input)
        inp.value = ""
        inp.add_class("-hidden")
        inp.can_focus = False
        self.focus()
        self._refresh_ui()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        if event.input.id != "tmux-input":
            return
        name = event.value.strip()
        if not name:
            self._close_input()
            return
        err = create_session(name)
        self._close_input()
        if err:
            self.status = f"create failed: {err}"
            self.status_is_error = True
            self.refresh_list()
            return
        self.status = f"created session '{name}'"
        self.status_is_error = False
        self.refresh_list()
        # Move cursor to the new session and fire attach.
        for i, s in enumerate(self.sessions):
            if s.name == name:
                self.cursor = i
                break
        self._refresh_ui()
        self.post_message(TmuxAttachRequested(name))
