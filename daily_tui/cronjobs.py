"""Cron Jobs tab: in-process scheduler with multi-trigger, cancel, per-job counters."""

from __future__ import annotations

import asyncio
import json
import shlex
import time
import uuid
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Dict, List, Optional, Set

from croniter import croniter
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Vertical
from textual.message import Message
from textual.widgets import Input, Static

from . import theme


SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]


@dataclass
class CronJob:
    id: str
    name: str
    schedule: str
    command: str
    enabled: bool = True
    created_at: float = field(default_factory=time.time)
    runs_success: int = 0
    runs_failed: int = 0
    last_run_at: Optional[float] = None
    last_status: str = "never"  # never | success | failed | running | cancelled
    # next_fire_after is the cron-time pointer the scheduler uses to avoid
    # double-firing. We advance it every time the job runs. Persisted so a
    # restart doesn't replay missed runs.
    next_fire_after: Optional[float] = None


class CronStore:
    """JSON-backed store for cron job definitions + persistent counters."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.jobs: List[CronJob] = []
        self._load()

    def _load(self) -> None:
        if not self.path.exists():
            return
        try:
            data = json.loads(self.path.read_text())
        except (json.JSONDecodeError, OSError):
            return
        self.jobs = [CronJob(**{k: v for k, v in item.items() if k in CronJob.__dataclass_fields__}) for item in data]

    def save(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.path.write_text(json.dumps([asdict(j) for j in self.jobs], indent=2))

    def add(self, name: str, schedule: str, command: str) -> CronJob:
        job = CronJob(
            id=uuid.uuid4().hex[:8],
            name=name,
            schedule=schedule,
            command=command,
            next_fire_after=time.time(),
        )
        self.jobs.append(job)
        self.save()
        return job

    def update(self, job_id: str, name: str, schedule: str, command: str) -> None:
        for j in self.jobs:
            if j.id == job_id:
                j.name = name
                j.schedule = schedule
                j.command = command
                # Reschedule from now when the cron expression changes.
                j.next_fire_after = time.time()
                self.save()
                return

    def delete(self, job_id: str) -> None:
        self.jobs = [j for j in self.jobs if j.id != job_id]
        self.save()

    def toggle_enabled(self, job_id: str) -> None:
        for j in self.jobs:
            if j.id == job_id:
                j.enabled = not j.enabled
                if j.enabled:
                    j.next_fire_after = time.time()
                self.save()
                return

    def record_run(self, job_id: str, success: bool, status: str) -> None:
        for j in self.jobs:
            if j.id == job_id:
                now = time.time()
                j.last_run_at = now
                j.last_status = status
                if success:
                    j.runs_success += 1
                else:
                    j.runs_failed += 1
                try:
                    itr = croniter(j.schedule, now)
                    j.next_fire_after = itr.get_next(float)
                except (ValueError, KeyError):
                    j.next_fire_after = None
                self.save()
                return

    def by_id(self, job_id: str) -> Optional[CronJob]:
        for j in self.jobs:
            if j.id == job_id:
                return j
        return None


def validate_cron(expr: str) -> Optional[str]:
    """Return None if valid, an error string otherwise."""
    try:
        croniter(expr, time.time())
        return None
    except (ValueError, KeyError) as exc:
        return str(exc)


class CronCountChanged(Message):
    def __init__(self, total: int, running: int) -> None:
        self.total = total
        self.running = running
        super().__init__()


class CronView(Vertical):
    """Cron jobs list + inline add/edit + runtime state."""

    DEFAULT_CSS = f"""
    CronView {{
        padding: 0 0;
    }}
    CronView Input {{
        border: tall {theme.SURFACE1};
        background: {theme.BASE};
        color: {theme.TEXT};
        margin-top: 1;
    }}
    CronView Input.-hidden {{
        display: none;
    }}
    CronView .help-bar {{
        color: {theme.OVERLAY};
        padding-top: 1;
    }}
    """

    BINDINGS = [
        Binding("up", "cursor_up", "up", show=False),
        Binding("down", "cursor_down", "down", show=False),
        Binding("a", "add", "add"),
        Binding("e", "edit", "edit"),
        Binding("d", "delete", "delete"),
        Binding("space", "toggle_enabled", "enable"),
        Binding("x", "mark", "mark"),
        Binding("t", "trigger_marked", "trigger marked"),
        Binding("enter", "trigger_focused", "run"),
        Binding("c", "cancel", "cancel"),
        Binding("r", "refresh", "refresh"),
        Binding("escape", "escape", "cancel", show=False),
    ]

    can_focus = True

    def __init__(self, store: CronStore) -> None:
        super().__init__()
        self.store = store
        self.cursor = 0
        self.marked: Set[str] = set()
        self.active_runs: Dict[str, asyncio.Task] = {}
        self.run_started: Dict[str, float] = {}
        self.mode: Optional[str] = None  # None | "add" | "edit"
        self.edit_id: Optional[str] = None
        self.status: str = ""
        self.status_is_error: bool = False
        self._spinner_phase = 0

    def compose(self) -> ComposeResult:
        yield Static("", id="cron-body")
        inp = Input(
            placeholder="name | */5 * * * * | echo hello",
            id="cron-input",
        )
        inp.add_class("-hidden")
        inp.can_focus = False
        yield inp
        yield Static("", id="cron-help", classes="help-bar")

    def on_mount(self) -> None:
        self._refresh_ui()
        # Spinner tick — cheap; only repaints while a run is active.
        self.set_interval(0.1, self._on_spinner_tick)
        # Scheduler tick — fires any enabled job whose cron matches.
        self.set_interval(2.0, self._on_schedule_tick)

    def is_inputting(self) -> bool:
        return self.mode is not None

    def title(self) -> str:
        total = len(self.store.jobs)
        running = len(self.active_runs)
        if total == 0:
            return "no jobs"
        if running:
            return f"{total} jobs · {running} running"
        return f"{total} job{'s' if total != 1 else ''}"

    def count(self) -> int:
        return len(self.store.jobs)

    def running_count(self) -> int:
        return len(self.active_runs)

    # ---- ticks ----

    def _on_spinner_tick(self) -> None:
        if not self.active_runs:
            return
        self._spinner_phase = (self._spinner_phase + 1) % len(SPINNER_FRAMES)
        self._refresh_ui()

    def _on_schedule_tick(self) -> None:
        now = time.time()
        fired = False
        for job in self.store.jobs:
            if not job.enabled:
                continue
            if job.id in self.active_runs:
                continue
            nxt = job.next_fire_after
            if nxt is None:
                try:
                    itr = croniter(job.schedule, now)
                    job.next_fire_after = itr.get_next(float)
                except (ValueError, KeyError):
                    job.next_fire_after = None
                continue
            if now >= nxt:
                self._start_run(job)
                fired = True
        if fired:
            self.store.save()
            self._refresh_ui()

    # ---- rendering ----

    def _refresh_ui(self) -> None:
        lines: list[str] = []
        total = len(self.store.jobs)
        running = len(self.active_runs)
        total_success = sum(j.runs_success for j in self.store.jobs)
        total_failed = sum(j.runs_failed for j in self.store.jobs)

        if total == 0:
            header = f"[{theme.TEXT}]Cron Jobs[/]   [{theme.OVERLAY}]0 jobs[/]"
        else:
            parts = [f"[{theme.TEXT}]Cron Jobs[/]", f"[{theme.PEACH}]{total} jobs[/]"]
            if running:
                parts.append(f"[{theme.GREEN}]{running} running[/]")
            parts.append(f"[{theme.GREEN}]✓ {total_success}[/]")
            parts.append(f"[{theme.RED}]✗ {total_failed}[/]")
            header = "   ".join(parts)
        lines.append(header)
        lines.append(f"[{theme.OVERLAY}]── JOBS {'─' * 60}[/]")

        if total == 0:
            lines.append(
                f"[{theme.OVERLAY}]  No jobs yet — press 'a' to add one "
                f"(format: name | schedule | command)[/]"
            )
        else:
            for i, job in enumerate(self.store.jobs):
                selected = (i == self.cursor) and not self.is_inputting()
                lines.append(self._render_row(job, selected))

        if self.marked:
            lines.append("")
            lines.append(
                f"[{theme.PEACH}]  {len(self.marked)} marked — press 't' to trigger all[/]"
            )

        if self.status:
            lines.append("")
            color = theme.RED if self.status_is_error else theme.GREEN
            lines.append(f"[{color}]  {self.status}[/]")

        self.query_one("#cron-body", Static).update("\n".join(lines))
        self.query_one("#cron-help", Static).update(self._help_bar())
        # Let the sidebar know counts have changed.
        self.post_message(CronCountChanged(total, running))

    def _render_row(self, job: CronJob, selected: bool) -> str:
        caret = f"[{theme.MAUVE}]▸[/]" if selected else " "
        running = job.id in self.active_runs
        marked = job.id in self.marked
        # Status glyph: spinner while running, last-run outcome otherwise.
        if running:
            glyph = f"[{theme.GREEN}]{SPINNER_FRAMES[self._spinner_phase]}[/]"
        elif job.last_status == "success":
            glyph = f"[{theme.GREEN}]✓[/]"
        elif job.last_status == "failed":
            glyph = f"[{theme.RED}]✗[/]"
        elif job.last_status == "cancelled":
            glyph = f"[{theme.YELLOW}]⊘[/]"
        else:
            glyph = f"[{theme.OVERLAY}]·[/]"

        enabled_glyph = (
            f"[{theme.GREEN}]●[/]"
            if job.enabled
            else f"[{theme.OVERLAY}]○[/]"
        )
        mark_box = f"[{theme.PEACH}][x][/]" if marked else f"[{theme.OVERLAY}][ ][/]"

        name_style = f"bold {theme.TEXT}" if selected else theme.TEXT
        name = f"[{name_style}]{job.name:<18}[/]"
        sched = f"[{theme.SUBTEXT0}]{job.schedule:<14}[/]"
        # Truncate command so the row stays readable.
        cmd = job.command if len(job.command) <= 30 else job.command[:27] + "…"
        cmd_fmt = f"[{theme.OVERLAY}]{cmd:<30}[/]"

        counts = (
            f"[{theme.GREEN}]✓{job.runs_success}[/] "
            f"[{theme.RED}]✗{job.runs_failed}[/]"
        )

        trailing = ""
        if running:
            elapsed = int(time.time() - self.run_started.get(job.id, time.time()))
            trailing = f"  [{theme.GREEN}]running {elapsed}s[/]"

        return (
            f"{caret} {glyph} {enabled_glyph} {mark_box} "
            f"{name} {sched} {cmd_fmt} {counts}{trailing}"
        )

    def _help_bar(self) -> str:
        if self.mode == "add":
            hints = [("enter", "create"), ("esc", "cancel")]
            hint_line = "format: name | schedule | command"
        elif self.mode == "edit":
            hints = [("enter", "save"), ("esc", "cancel")]
            hint_line = "format: name | schedule | command"
        else:
            hints = [
                ("↑↓", "nav"),
                ("enter", "run"),
                ("a", "add"),
                ("e", "edit"),
                ("d", "del"),
                ("space", "enable"),
                ("x", "mark"),
                ("t", "run marked"),
                ("c", "cancel"),
            ]
            hint_line = ""
        parts = []
        for k, label in hints:
            parts.append(
                f"[{theme.TEXT} on {theme.SURFACE}] {k} [/] "
                f"[{theme.OVERLAY}]{label}[/]"
            )
        bar = "  ".join(parts)
        if hint_line:
            bar += f"\n  [{theme.OVERLAY}]{hint_line}[/]"
        return bar

    # ---- input parsing ----

    @staticmethod
    def _parse_input(text: str) -> Optional[tuple[str, str, str]]:
        parts = [p.strip() for p in text.split("|")]
        if len(parts) < 3:
            return None
        name = parts[0]
        schedule = parts[1]
        command = "|".join(parts[2:]).strip()
        if not name or not schedule or not command:
            return None
        return name, schedule, command

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
        if self.cursor < len(self.store.jobs) - 1:
            self.cursor += 1
            self._refresh_ui()

    def action_add(self) -> None:
        if self.is_inputting():
            return
        self.mode = "add"
        self.status = ""
        inp = self.query_one("#cron-input", Input)
        inp.value = ""
        inp.placeholder = "name | */5 * * * * | echo hello"
        inp.remove_class("-hidden")
        inp.can_focus = True
        inp.focus()
        self._refresh_ui()

    def action_edit(self) -> None:
        if self.is_inputting() or not self.store.jobs:
            return
        job = self.store.jobs[self.cursor]
        self.mode = "edit"
        self.edit_id = job.id
        self.status = ""
        inp = self.query_one("#cron-input", Input)
        inp.value = f"{job.name} | {job.schedule} | {job.command}"
        inp.placeholder = "name | schedule | command"
        inp.remove_class("-hidden")
        inp.can_focus = True
        inp.focus()
        self._refresh_ui()

    def action_delete(self) -> None:
        if self.is_inputting() or not self.store.jobs:
            return
        job = self.store.jobs[self.cursor]
        # Cancel any in-flight run before removing the row.
        if job.id in self.active_runs:
            self.active_runs[job.id].cancel()
        self.marked.discard(job.id)
        self.store.delete(job.id)
        if self.cursor >= len(self.store.jobs) and self.cursor > 0:
            self.cursor -= 1
        self.status = f"deleted '{job.name}'"
        self.status_is_error = False
        self._refresh_ui()

    def action_toggle_enabled(self) -> None:
        if self.is_inputting() or not self.store.jobs:
            return
        job = self.store.jobs[self.cursor]
        self.store.toggle_enabled(job.id)
        self._refresh_ui()

    def action_mark(self) -> None:
        if self.is_inputting() or not self.store.jobs:
            return
        job = self.store.jobs[self.cursor]
        if job.id in self.marked:
            self.marked.remove(job.id)
        else:
            self.marked.add(job.id)
        self._refresh_ui()

    def action_trigger_focused(self) -> None:
        if self.is_inputting() or not self.store.jobs:
            return
        self._start_run(self.store.jobs[self.cursor])
        self._refresh_ui()

    def action_trigger_marked(self) -> None:
        if self.is_inputting():
            return
        targets = [j for j in self.store.jobs if j.id in self.marked]
        if not targets:
            self.status = "no jobs marked — press 'x' to mark first"
            self.status_is_error = True
            self._refresh_ui()
            return
        for job in targets:
            self._start_run(job)
        self.status = f"triggered {len(targets)} job{'s' if len(targets) != 1 else ''}"
        self.status_is_error = False
        self.marked.clear()
        self._refresh_ui()

    def action_cancel(self) -> None:
        if self.is_inputting() or not self.store.jobs:
            return
        job = self.store.jobs[self.cursor]
        task = self.active_runs.get(job.id)
        if task is None:
            self.status = f"'{job.name}' is not running"
            self.status_is_error = True
        else:
            task.cancel()
            self.status = f"cancelling '{job.name}'…"
            self.status_is_error = False
        self._refresh_ui()

    def action_refresh(self) -> None:
        if self.is_inputting():
            return
        self.status = ""
        self._refresh_ui()

    def action_escape(self) -> None:
        if self.is_inputting():
            self._close_input()

    def _close_input(self) -> None:
        self.mode = None
        self.edit_id = None
        inp = self.query_one("#cron-input", Input)
        inp.value = ""
        inp.add_class("-hidden")
        inp.can_focus = False
        self.focus()
        self._refresh_ui()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        if event.input.id != "cron-input":
            return
        parsed = self._parse_input(event.value)
        if parsed is None:
            self.status = "invalid — use: name | schedule | command"
            self.status_is_error = True
            self._close_input()
            return
        name, schedule, command = parsed
        err = validate_cron(schedule)
        if err:
            self.status = f"invalid cron '{schedule}': {err}"
            self.status_is_error = True
            self._close_input()
            return
        if self.mode == "add":
            self.store.add(name, schedule, command)
            self.cursor = len(self.store.jobs) - 1
            self.status = f"added '{name}'"
        elif self.mode == "edit" and self.edit_id:
            self.store.update(self.edit_id, name, schedule, command)
            self.status = f"updated '{name}'"
        self.status_is_error = False
        self._close_input()

    # ---- runner ----

    def _start_run(self, job: CronJob) -> None:
        if job.id in self.active_runs:
            return
        job.last_status = "running"
        self.run_started[job.id] = time.time()
        task = asyncio.create_task(self._run_async(job.id, job.command))
        self.active_runs[job.id] = task

    async def _run_async(self, job_id: str, command: str) -> None:
        status = "failed"
        success = False
        try:
            proc = await asyncio.create_subprocess_shell(
                command,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
            )
            try:
                _stdout, _ = await proc.communicate()
                if proc.returncode == 0:
                    success = True
                    status = "success"
                else:
                    status = "failed"
            except asyncio.CancelledError:
                proc.kill()
                await proc.wait()
                status = "cancelled"
                raise
        except asyncio.CancelledError:
            status = "cancelled"
        except Exception:
            status = "failed"
        finally:
            self.store.record_run(job_id, success, status)
            self.active_runs.pop(job_id, None)
            self.run_started.pop(job_id, None)
            self._refresh_ui()
