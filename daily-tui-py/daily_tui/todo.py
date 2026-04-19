"""Todo tab: JSON-backed task list with add/toggle/edit/delete."""

from __future__ import annotations

import json
import time
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import List, Optional

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Container, Vertical
from textual.message import Message
from textual.widgets import Input, Static

from . import theme


@dataclass
class Task:
    id: str
    text: str
    done: bool = False
    created_at: float = field(default_factory=time.time)
    updated_at: Optional[float] = None


class TodoStore:
    """Loads/saves tasks to a JSON file."""

    def __init__(self, path: Path) -> None:
        self.path = path
        self.tasks: List[Task] = []
        self._load()

    def _load(self) -> None:
        if not self.path.exists():
            return
        try:
            raw = json.loads(self.path.read_text())
        except (json.JSONDecodeError, OSError):
            return
        self.tasks = [
            Task(
                id=item["id"],
                text=item["text"],
                done=item.get("done", False),
                created_at=item.get("created_at", time.time()),
                updated_at=item.get("updated_at"),
            )
            for item in raw
        ]

    def _save(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.path.write_text(json.dumps([asdict(t) for t in self.tasks], indent=2))

    def add(self, text: str) -> Task:
        task = Task(id=f"{time.time_ns()}", text=text)
        self.tasks.append(task)
        self._save()
        return task

    def toggle(self, task_id: str) -> None:
        for t in self.tasks:
            if t.id == task_id:
                t.done = not t.done
                self._save()
                return

    def update_text(self, task_id: str, text: str) -> None:
        for t in self.tasks:
            if t.id == task_id:
                t.text = text
                t.updated_at = time.time()
                self._save()
                return

    def delete(self, task_id: str) -> None:
        self.tasks = [t for t in self.tasks if t.id != task_id]
        self._save()


class TodoCountChanged(Message):
    def __init__(self, pending: int, done: int) -> None:
        self.pending = pending
        self.done = done
        super().__init__()


class TodoView(Vertical):
    """Todo tab widget: header, task list, inline input."""

    DEFAULT_CSS = f"""
    TodoView {{
        padding: 0 0;
    }}
    TodoView .todo-header {{
        color: {theme.TEXT};
        padding-bottom: 1;
    }}
    TodoView .todo-header-sub {{
        color: {theme.OVERLAY};
    }}
    TodoView .todo-divider {{
        color: {theme.OVERLAY};
    }}
    TodoView .todo-row {{
        padding: 0 0;
    }}
    TodoView .todo-row.-selected {{
        color: {theme.TEXT};
        text-style: bold;
        background: {theme.SURFACE};
    }}
    TodoView .todo-row.-done {{
        color: {theme.OVERLAY};
        text-style: strike;
    }}
    TodoView .todo-empty {{
        color: {theme.OVERLAY};
    }}
    TodoView Input {{
        border: tall {theme.SURFACE1};
        background: {theme.BASE};
        color: {theme.TEXT};
        margin-top: 1;
    }}
    TodoView Input.-hidden {{
        display: none;
    }}
    TodoView .help-bar {{
        color: {theme.OVERLAY};
        padding-top: 1;
    }}
    """

    BINDINGS = [
        Binding("up", "cursor_up", "up", show=False),
        Binding("down", "cursor_down", "down", show=False),
        Binding("space", "toggle", "done"),
        Binding("a", "add", "add"),
        Binding("e", "edit", "edit"),
        Binding("d", "delete", "delete"),
        Binding("escape", "cancel", "cancel", show=False),
    ]

    can_focus = True

    def __init__(self, store: TodoStore) -> None:
        super().__init__()
        self.store = store
        self.cursor = 0
        self.mode: Optional[str] = None  # None | "add" | "edit"
        self.edit_id: Optional[str] = None

    def compose(self) -> ComposeResult:
        yield Static("", id="todo-body", classes="todo-body")
        inp = Input(placeholder="Add a task...", id="todo-input")
        inp.add_class("-hidden")
        inp.can_focus = False
        yield inp
        yield Static("", id="todo-help", classes="help-bar")

    def on_mount(self) -> None:
        self._refresh()

    # ---- ordered list (pending first, then completed) ----

    def _ordered(self) -> List[Task]:
        pending = [t for t in self.store.tasks if not t.done]
        done = [t for t in self.store.tasks if t.done]
        return pending + done

    def _pending_count(self) -> int:
        return sum(1 for t in self.store.tasks if not t.done)

    def _done_count(self) -> int:
        return sum(1 for t in self.store.tasks if t.done)

    def title(self) -> str:
        total = len(self.store.tasks)
        pending = self._pending_count()
        done = self._done_count()
        if total == 0:
            return "no tasks"
        if pending == 0:
            return "all done"
        return f"{pending} pending · {done} done"

    def is_inputting(self) -> bool:
        return self.mode is not None

    # ---- rendering ----

    def _refresh(self) -> None:
        ordered = self._ordered()
        if self.cursor >= len(ordered):
            self.cursor = max(0, len(ordered) - 1)

        lines: list[str] = []
        total = len(ordered)
        pending = self._pending_count()
        done = self._done_count()

        header = (
            f"[bold {theme.TEXT}]Today's Tasks[/] "
            f"[{theme.SUBTEXT0}]{pending} open[/] "
            f"[{theme.OVERLAY}]·[/] "
            f"[{theme.SUBTEXT0}]{done} done[/]"
        )
        lines.append(header)
        lines.append(f"[{theme.OVERLAY}]── TASKS {'─' * 58}[/]")

        if total == 0:
            lines.append(f"[{theme.OVERLAY}]  No tasks yet — press 'a' to add one[/]")
        else:
            for i, task in enumerate(ordered):
                selected = (i == self.cursor) and not self.is_inputting()
                lines.append(self._render_row(task, i + 1, selected))

        if total > 0 and pending > 0 and done > 0:
            pct = int(100 * done / total) if total else 0
            filled = max(0, min(40, int(40 * done / total))) if total else 0
            bar = (
                f"[{theme.GREEN}]{'▓' * filled}[/]"
                f"[{theme.OVERLAY}]{'░' * (40 - filled)}[/]"
                f"  [{theme.OVERLAY}]{pct}%[/]"
            )
            lines.append("")
            lines.append(bar)

        self.query_one("#todo-body", Static).update("\n".join(lines))
        self.query_one("#todo-help", Static).update(self._help_bar())
        self.post_message(TodoCountChanged(pending, done))

    def _render_row(self, task: Task, index: int, selected: bool) -> str:
        caret = f"[{theme.MAUVE}]▸[/]" if selected else " "
        box = "☑" if task.done else "☐"
        num = f"[{theme.OVERLAY}]#{index:02d}[/]"
        if task.done:
            text = f"[{theme.OVERLAY} strike]{task.text}[/]"
            box_style = f"[{theme.OVERLAY}]{box}[/]"
        elif selected:
            text = f"[bold {theme.TEXT}]{task.text}[/]"
            box_style = f"[{theme.TEXT}]{box}[/]"
        else:
            text = f"[{theme.TEXT}]{task.text}[/]"
            box_style = f"[{theme.TEXT}]{box}[/]"
        row = f"{caret}  {box_style}  {text}"
        pad_target = 80
        plain_len = 4 + 1 + 2 + len(task.text)
        pad = max(1, pad_target - plain_len)
        return f"{row}{' ' * pad}{num}"

    def _help_bar(self) -> str:
        if self.mode == "add":
            hints = [("enter", "confirm"), ("esc", "cancel")]
        elif self.mode == "edit":
            hints = [("enter", "save"), ("esc", "cancel")]
        else:
            hints = [
                ("↑↓", "navigate"),
                ("space", "toggle"),
                ("a", "add"),
                ("e", "edit"),
                ("d", "delete"),
            ]
        parts = []
        for k, label in hints:
            parts.append(
                f"[{theme.TEXT} on {theme.SURFACE}] {k} [/] "
                f"[{theme.OVERLAY}]{label}[/]"
            )
        return "  ".join(parts)

    # ---- actions ----

    def action_cursor_up(self) -> None:
        if self.is_inputting():
            return
        if self.cursor > 0:
            self.cursor -= 1
            self._refresh()

    def action_cursor_down(self) -> None:
        if self.is_inputting():
            return
        if self.cursor < len(self._ordered()) - 1:
            self.cursor += 1
            self._refresh()

    def action_toggle(self) -> None:
        if self.is_inputting():
            return
        ordered = self._ordered()
        if not ordered:
            return
        self.store.toggle(ordered[self.cursor].id)
        self._refresh()

    def action_add(self) -> None:
        if self.is_inputting():
            return
        self.mode = "add"
        inp = self.query_one("#todo-input", Input)
        inp.placeholder = "Add a task..."
        inp.value = ""
        inp.remove_class("-hidden")
        inp.can_focus = True
        inp.focus()
        self._refresh()

    def action_edit(self) -> None:
        if self.is_inputting():
            return
        ordered = self._ordered()
        if not ordered:
            return
        task = ordered[self.cursor]
        self.mode = "edit"
        self.edit_id = task.id
        inp = self.query_one("#todo-input", Input)
        inp.placeholder = "Edit task..."
        inp.value = task.text
        inp.remove_class("-hidden")
        inp.can_focus = True
        inp.focus()
        self._refresh()

    def action_delete(self) -> None:
        if self.is_inputting():
            return
        ordered = self._ordered()
        if not ordered:
            return
        self.store.delete(ordered[self.cursor].id)
        self._refresh()

    def action_cancel(self) -> None:
        if not self.is_inputting():
            return
        self._close_input()

    def _close_input(self) -> None:
        self.mode = None
        self.edit_id = None
        inp = self.query_one("#todo-input", Input)
        inp.value = ""
        inp.add_class("-hidden")
        inp.can_focus = False
        self.focus()
        self._refresh()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        if event.input.id != "todo-input":
            return
        text = event.value.strip()
        if text:
            if self.mode == "add":
                self.store.add(text)
                self.cursor = len(self._ordered()) - 1
            elif self.mode == "edit" and self.edit_id:
                self.store.update_text(self.edit_id, text)
        self._close_input()
