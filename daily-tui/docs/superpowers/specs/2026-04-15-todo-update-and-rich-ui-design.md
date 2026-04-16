# Todo Update Functionality & Rich UI Design

**Date:** 2026-04-15
**Issue:** [#2 — add update functionality to todo app](https://github.com/DevOpsAIguru123/productivity-tools/issues/2)

## Summary

Add edit/update functionality to the todo app and overhaul the UI with a sectioned layout, progress bar, task counts, and improved visual hierarchy.

## 1. Store Layer Changes

### New Field: `UpdatedAt`

Add `UpdatedAt *time.Time` to the `Task` struct. Pointer type so it's `nil` for never-edited tasks and omitted from JSON when empty.

```go
type Task struct {
    ID        string     `json:"id"`
    Text      string     `json:"text"`
    Done      bool       `json:"done"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
```

### New Method: `Store.Update`

```go
func (s *Store) Update(id, text string)
```

Finds task by ID, replaces `Text`, sets `UpdatedAt` to `time.Now()`, saves. No-op if ID not found.

## 2. Model State Changes

### New Fields

```go
type Model struct {
    store      *Store
    tasks      []Task
    cursor     int
    adding     bool
    editing    bool       // true when editing an existing task
    editTaskID string     // ID of the task being edited
    input      textinput.Model
}
```

`adding` and `editing` are mutually exclusive — both use the same `input` component.

### Edit Mode Transitions

| Key     | Context                        | Action                                                                 |
|---------|--------------------------------|------------------------------------------------------------------------|
| `e`     | Task selected, not adding/editing, list non-empty | Enter edit mode: `editing=true`, `editTaskID=tasks[cursor].ID`, pre-fill input with task text, focus input. No-op if task list is empty. |
| `Enter` | In edit mode                   | If text non-empty and changed: `Store.Update(editTaskID, text)`. Exit edit mode, blur input, clear value |
| `Esc`   | In edit mode                   | Cancel: blur input, clear value, `editing=false`                       |

### Cursor Behavior in Sectioned View

- Tasks are displayed in two sections (pending first, completed second) but the cursor indexes into a **single flat list** formed by concatenating pending + completed tasks
- Section headers are visual only — not selectable
- When toggling a task from pending→done, the task moves to the completed section; cursor stays at same index (adjusts if out of bounds)

## 3. View Layout

```
  Today's Tasks                         3/5 done
  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░░░  60%

  ── Pending (2) ──────────────────────────
  ▸ ☐  Fix login bug
    ☐  Write tests

  ── Completed (3) ────────────────────────
    ☑  Set up CI
    ☑  Add README
    ☑  Deploy v1

  ↑↓ nav • space done • e edit • d del • a add
```

### Layout Components

1. **Header line** — "Today's Tasks" in dimmed + "N/M done" right-aligned in peach badge color
2. **Progress bar** — 30 chars wide. Filled blocks (`▓`) in green, empty (`░`) in overlay/dimmed. Shows percentage.
3. **Pending section** — "── Pending (N) ──────" in normal text color. Lists undone tasks.
4. **Completed section** — "── Completed (N) ──────" in dimmed. Lists done tasks in dimmed text.
5. **Empty states** — "No pending tasks — press 'a' to add one" / "No completed tasks yet" in dimmed
6. **Input area** — Appears below sections when adding or editing. When editing, placeholder shows "Edit task..."
7. **Help bar** — Context-sensitive:
   - Normal: `↑↓ nav • space done • e edit • d del • a add`
   - Adding: `enter confirm • esc cancel`
   - Editing: `enter save • esc cancel`

### Theme Additions

Add to `internal/theme/theme.go`:

- `ProgressFilled` — green foreground (reuse `Green` color)
- `ProgressEmpty` — overlay foreground
- `SectionHeader` — text color, bold

## 4. Task Ordering

Within each section, tasks appear in insertion order (same as current behavior — order from `Store.All()`). The view splits them:

```go
var pending, completed []Task
for _, t := range m.tasks {
    if t.Done {
        completed = append(completed, t)
    } else {
        pending = append(pending, t)
    }
}
// Display order: pending first, then completed
// Cursor maps to: index 0..len(pending)-1 = pending, len(pending)..end = completed
```

## 5. Testing

### Store Tests

- `TestUpdate` — update text of existing task, verify text changed and `UpdatedAt` is set
- `TestUpdateNonExistent` — update non-existent ID, verify no crash and no changes
- `TestUpdatePersistence` — update, reload store, verify change survived

### Model Tests

- `TestEditMode` — press `e`, verify `editing=true` and input pre-filled
- `TestEditSave` — enter edit mode, type new text, press Enter, verify task updated
- `TestEditCancel` — enter edit mode, press Esc, verify task unchanged
- `TestEditEmptyText` — enter edit mode, clear text, press Enter, verify task unchanged (no empty text allowed)
- `TestSectionedCursor` — verify cursor navigates across pending and completed sections correctly

## 6. Files Modified

| File | Changes |
|------|---------|
| `internal/todo/store.go` | Add `UpdatedAt` field to `Task`, add `Update` method |
| `internal/todo/model.go` | Add `editing`/`editTaskID` fields, edit key handling, sectioned `View()` with progress bar |
| `internal/theme/theme.go` | Add `ProgressFilled`, `ProgressEmpty`, `SectionHeader` styles |
| `internal/todo/store_test.go` | Add `TestUpdate`, `TestUpdateNonExistent`, `TestUpdatePersistence` |
| `internal/todo/model_test.go` | Add edit mode tests, sectioned cursor test |

## 7. Non-Goals

- Priority levels or tags (future enhancement)
- Due dates or reminders
- Undo/redo
- Drag-to-reorder
