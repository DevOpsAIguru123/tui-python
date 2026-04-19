"""Claude Commands tab: discover slash-commands and run `claude -p ...` async."""

from __future__ import annotations

import asyncio
import os
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Optional

import yaml
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Vertical
from textual.message import Message
from textual.widgets import Static

from . import theme


@dataclass
class Command:
    name: str
    prompt: str
    source: str  # "builtin" | "config" | "user" | "project"
    extra_args: List[str] = field(default_factory=list)
    path: Optional[str] = None


@dataclass
class RunState:
    status: str = "idle"  # "idle" | "running" | "done"
    output: str = ""
    err: str = ""
    start_at: float = 0.0
    finish_at: float = 0.0


def _builtins() -> List[Command]:
    return [
        Command(name="hello", prompt="hello", source="builtin"),
        Command(
            name="network-monitor",
            prompt="run this skill /network-monitor and store results in llm wiki",
            source="builtin",
            extra_args=["--dangerously-skip-permissions"],
        ),
    ]


def _from_config(config_path: Path) -> List[Command]:
    if not config_path.exists():
        return []
    try:
        data = yaml.safe_load(config_path.read_text()) or {}
    except (yaml.YAMLError, OSError):
        return []
    entries = (data.get("claude_code") or {}).get("commands") or []
    out: List[Command] = []
    for entry in entries:
        name = (entry.get("name") or "").strip()
        prompt = (entry.get("prompt") or "").strip()
        if not name or not prompt:
            continue
        out.append(
            Command(
                name=name,
                prompt=prompt,
                source="config",
                extra_args=list(entry.get("extra_args") or []),
            )
        )
    return out


def _scan_commands_dir(root: Path, source: str) -> List[Command]:
    if not root.is_dir():
        return []
    cmds: List[Command] = []
    for md in sorted(root.rglob("*.md")):
        rel = md.relative_to(root).with_suffix("")
        name = ":".join(rel.parts)
        cmds.append(
            Command(name=name, prompt=f"/{name}", source=source, path=str(md))
        )
    cmds.sort(key=lambda c: c.name)
    return cmds


def discover_commands(config_path: Path) -> List[Command]:
    out: List[Command] = []
    out.extend(_builtins())
    out.extend(_from_config(config_path))
    home = Path(os.path.expanduser("~"))
    out.extend(_scan_commands_dir(home / ".claude" / "commands", "user"))
    out.extend(_scan_commands_dir(Path.cwd() / ".claude" / "commands", "project"))
    return out


class ClaudeView(Vertical):
    """Claude commands tab: pick an entry, Enter runs `claude -p`."""

    DEFAULT_CSS = f"""
    ClaudeView {{
        padding: 0 0;
    }}
    ClaudeView .help-bar {{
        color: {theme.OVERLAY};
        padding-top: 1;
    }}
    """

    BINDINGS = [
        Binding("up", "cursor_up", "up", show=False),
        Binding("down", "cursor_down", "down", show=False),
        Binding("enter", "run", "run"),
        Binding("escape", "clear", "clear", show=False),
    ]

    RUN_TIMEOUT_SEC = 5 * 60
    can_focus = True

    def __init__(self, config_path: Path) -> None:
        super().__init__()
        self.commands = discover_commands(config_path)
        self.runs: Dict[str, RunState] = {}
        self.cursor = 0
        self._tasks: Dict[str, asyncio.Task] = {}

    def compose(self) -> ComposeResult:
        yield Static("", id="claude-body")
        yield Static("", id="claude-help", classes="help-bar")

    def on_mount(self) -> None:
        self._refresh()
        self.set_interval(1.0, self._tick_running)

    def is_inputting(self) -> bool:
        return False

    def title(self) -> str:
        running = sum(1 for r in self.runs.values() if r.status == "running")
        total = len(self.commands)
        if running > 1:
            return f"{running} running"
        if running == 1:
            name = next(
                (c.name for c in self.commands if self.runs.get(c.name) and self.runs[c.name].status == "running"),
                "",
            )
            return f"running · {name}"
        suffix = "s" if total != 1 else ""
        if self.runs:
            return f"{total} command{suffix} · idle"
        return f"{total} command{suffix}"

    # ---- tick (update elapsed time for running runs) ----

    def _tick_running(self) -> None:
        if any(r.status == "running" for r in self.runs.values()):
            self._refresh()

    # ---- rendering ----

    def _refresh(self) -> None:
        lines: list[str] = []
        lines.append(self._render_header())
        lines.append("")
        lines.append(f"[{theme.OVERLAY}]── COMMANDS {'─' * 58}[/]")

        if not self.commands:
            lines.append(f"[{theme.OVERLAY}]  no commands found[/]")
        else:
            for i, c in enumerate(self.commands):
                lines.append(self._render_row(c, i == self.cursor))

        lines.append("")
        lines.append(self._render_output())

        self.query_one("#claude-body", Static).update("\n".join(lines))
        self.query_one("#claude-help", Static).update(self._help_bar())

    def _render_header(self) -> str:
        running = [
            c for c in self.commands
            if (r := self.runs.get(c.name)) and r.status == "running"
        ]
        if len(running) == 1:
            c = running[0]
            elapsed = int(time.time() - self.runs[c.name].start_at)
            return (
                f"[{theme.GREEN}]●[/] [{theme.TEXT}]running[/] "
                f"[{theme.GREEN}]{c.name}[/]  "
                f"[{theme.OVERLAY}]({elapsed}s)[/]"
            )
        if len(running) > 1:
            return (
                f"[{theme.GREEN}]●[/] "
                f"[{theme.TEXT}]{len(running)} commands running concurrently[/]"
            )
        return (
            f"[{theme.OVERLAY}]pick a command and press[/] "
            f"[{theme.GREEN}]enter[/]"
            f"[{theme.OVERLAY}] — multiple commands can run in parallel[/]"
        )

    def _render_row(self, cmd: Command, selected: bool) -> str:
        caret = f"[{theme.MAUVE}]▸[/]" if selected else " "
        name_style = f"bold {theme.TEXT}" if selected else theme.TEXT
        name = f"[{name_style}]{cmd.name}[/]"
        tag = f"[{theme.SUBTEXT0} on {theme.SURFACE}] {cmd.source} [/]"
        prompt = f"[{theme.OVERLAY}]{cmd.prompt}[/]"
        row = f"{caret}  {name}  {tag}  {prompt}"
        if cmd.extra_args:
            flags = " ".join(cmd.extra_args)
            color = theme.RED if any("dangerously" in a for a in cmd.extra_args) else theme.OVERLAY
            row += f"  [{color}]{flags}[/]"
        badge = self._render_badge(cmd.name)
        if badge:
            row += f"  {badge}"
        return row

    def _render_badge(self, name: str) -> str:
        r = self.runs.get(name)
        if not r:
            return ""
        if r.status == "running":
            elapsed = int(time.time() - r.start_at)
            return f"[{theme.GREEN}]● running {elapsed}s[/]"
        if r.status == "done":
            dur = r.finish_at - r.start_at
            if r.err:
                return f"[{theme.RED}]✗ {dur:.2f}s failed[/]"
            return f"[{theme.GREEN}]✓[/] [{theme.OVERLAY}]{dur:.2f}s[/]"
        return ""

    def _render_output(self) -> str:
        if not self.commands:
            return ""
        cmd = self.commands[self.cursor]
        r = self.runs.get(cmd.name)
        if not r:
            return ""
        if r.status == "running":
            return (
                f"[{theme.OVERLAY}]  waiting for claude…  "
                f"(other commands may also be running — ↑↓ to check)[/]"
            )
        if r.status == "done":
            parts = [f"[{theme.OVERLAY}]── OUTPUT {'─' * 60}[/]"]
            if r.err:
                parts.append(f"[{theme.RED}]  {r.err}[/]")
            if r.output:
                parts.append(r.output)
            return "\n".join(parts)
        return ""

    def _help_bar(self) -> str:
        hints = [("↑↓", "navigate"), ("enter", "run")]
        cmd = self.commands[self.cursor] if self.commands else None
        if cmd and (r := self.runs.get(cmd.name)) and r.status == "done":
            hints.append(("esc", "clear"))
        parts = []
        for k, label in hints:
            parts.append(
                f"[{theme.TEXT} on {theme.SURFACE}] {k} [/] "
                f"[{theme.OVERLAY}]{label}[/]"
            )
        return "  ".join(parts)

    # ---- actions ----

    def action_cursor_up(self) -> None:
        if self.cursor > 0:
            self.cursor -= 1
            self._refresh()

    def action_cursor_down(self) -> None:
        if self.cursor < len(self.commands) - 1:
            self.cursor += 1
            self._refresh()

    def action_run(self) -> None:
        if not self.commands:
            return
        cmd = self.commands[self.cursor]
        existing = self.runs.get(cmd.name)
        if existing and existing.status == "running":
            return
        self.runs[cmd.name] = RunState(status="running", start_at=time.time())
        self._refresh()
        task = asyncio.create_task(self._run_async(cmd))
        self._tasks[cmd.name] = task

    def action_clear(self) -> None:
        if not self.commands:
            return
        cmd = self.commands[self.cursor]
        r = self.runs.get(cmd.name)
        if r and r.status == "done":
            del self.runs[cmd.name]
            self._refresh()

    async def _run_async(self, cmd: Command) -> None:
        argv = ["claude", "-p", cmd.prompt, *cmd.extra_args]
        output = ""
        err = ""
        try:
            proc = await asyncio.create_subprocess_exec(
                *argv,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
            )
            try:
                stdout, _ = await asyncio.wait_for(
                    proc.communicate(), timeout=self.RUN_TIMEOUT_SEC
                )
                output = stdout.decode("utf-8", errors="replace")
                if proc.returncode != 0:
                    err = f"exit status {proc.returncode}"
            except asyncio.TimeoutError:
                proc.kill()
                await proc.wait()
                err = f"timeout after {self.RUN_TIMEOUT_SEC}s"
        except FileNotFoundError:
            err = "`claude` binary not found on PATH"
        except Exception as exc:  # pragma: no cover - defensive
            err = f"{type(exc).__name__}: {exc}"

        r = self.runs.get(cmd.name)
        if r is None:
            r = RunState()
            self.runs[cmd.name] = r
        r.status = "done"
        r.output = output
        r.err = err
        r.finish_at = time.time()
        self._refresh()
