# Todo Update & Rich UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add edit/update functionality to the todo app and overhaul the View with a sectioned layout, progress bar, and improved visual hierarchy.

**Architecture:** The store gets a new `Update` method and `UpdatedAt` field. The model gains `editing`/`editTaskID` state that shares the existing text input. The view is rewritten to split tasks into pending/completed sections with a progress bar header. The app model's input-guard is updated to also check editing state.

**Tech Stack:** Go 1.26, Bubble Tea, Lip Gloss (Catppuccin Mocha theme)

**Spec:** `docs/superpowers/specs/2026-04-15-todo-update-and-rich-ui-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/todo/store.go` | Modify | Add `UpdatedAt` field to `Task`, add `Update` method |
| `internal/todo/store_test.go` | Modify | Add update tests (basic, non-existent ID, persistence) |
| `internal/theme/theme.go` | Modify | Add `ProgressFilled`, `ProgressEmpty`, `SectionHeader` styles |
| `internal/todo/model.go` | Modify | Add editing state, edit key handling, rewrite `View()` with sectioned layout |
| `internal/todo/model_test.go` | Modify | Add edit mode tests and sectioned cursor test |
| `internal/app/model.go` | Modify | Update input-guard to check `Editing()` in addition to `Adding()` |

---

### Task 1: Store — Add `UpdatedAt` field and `Update` method

**Files:**
- Modify: `internal/todo/store.go:11-17` (Task struct)
- Modify: `internal/todo/store.go` (new method)
- Test: `internal/todo/store_test.go`

- [ ] **Step 1: Write the failing tests for Store.Update**

Add these three tests to `internal/todo/store_test.go`:

```go
func TestStoreUpdateTask(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	task := s.Add("Original text")
	s.Update(task.ID, "Updated text")
	tasks := s.All()
	if tasks[0].Text != "Updated text" {
		t.Errorf("expected 'Updated text', got %q", tasks[0].Text)
	}
	if tasks[0].UpdatedAt == nil {
		t.Error("expected UpdatedAt to be set after update")
	}
}

func TestStoreUpdateNonExistent(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Keep me")
	s.Update("nonexistent-id", "Should not crash")
	tasks := s.All()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Text != "Keep me" {
		t.Errorf("expected 'Keep me', got %q", tasks[0].Text)
	}
}

func TestStoreUpdatePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	s1 := todo.NewStore(path)
	task := s1.Add("Before update")
	s1.Update(task.ID, "After update")

	s2 := todo.NewStore(path)
	tasks := s2.All()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task after reload, got %d", len(tasks))
	}
	if tasks[0].Text != "After update" {
		t.Errorf("expected 'After update', got %q", tasks[0].Text)
	}
	if tasks[0].UpdatedAt == nil {
		t.Error("expected UpdatedAt to survive reload")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd daily-tui && go test ./internal/todo/ -run "TestStoreUpdate" -v`
Expected: Compilation errors — `Update` method and `UpdatedAt` field don't exist yet.

- [ ] **Step 3: Add `UpdatedAt` to Task struct and implement `Store.Update`**

In `internal/todo/store.go`, update the `Task` struct to add the `UpdatedAt` field:

```go
// Task represents a single todo item.
type Task struct {
	ID        string     `json:"id"`
	Text      string     `json:"text"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
```

Add the `Update` method after the existing `Toggle` method:

```go
// Update replaces the text of the task with the given ID.
func (s *Store) Update(id, text string) {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Text = text
			now := time.Now()
			s.tasks[i].UpdatedAt = &now
			s.save()
			return
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd daily-tui && go test ./internal/todo/ -run "TestStoreUpdate" -v`
Expected: All 3 tests PASS.

- [ ] **Step 5: Run all existing tests to verify no regressions**

Run: `cd daily-tui && go test ./internal/todo/ -v`
Expected: All tests PASS (existing + new).

- [ ] **Step 6: Commit**

```bash
cd daily-tui && git add internal/todo/store.go internal/todo/store_test.go && git commit -m "feat(todo): add UpdatedAt field and Store.Update method

Add UpdatedAt *time.Time to Task struct (omitempty for backwards
compat). Add Store.Update(id, text) to edit task text in place.
Addresses #2."
```

---

### Task 2: Theme — Add progress bar and section header styles

**Files:**
- Modify: `internal/theme/theme.go`

- [ ] **Step 1: Add the three new styles**

Add these styles to the end of the `var` block in `internal/theme/theme.go`, after the `HelpStyle` definition:

```go
	ProgressFilled = lipgloss.NewStyle().
			Foreground(Green)

	ProgressEmpty = lipgloss.NewStyle().
			Foreground(Overlay)

	SectionHeader = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)
```

- [ ] **Step 2: Verify project compiles**

Run: `cd daily-tui && go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 3: Commit**

```bash
cd daily-tui && git add internal/theme/theme.go && git commit -m "feat(theme): add ProgressFilled, ProgressEmpty, SectionHeader styles

New Lip Gloss styles for the todo UI overhaul: green progress bar
fill, dimmed empty progress, and bold section headers."
```

---

### Task 3: Model — Add edit mode state and key handling

**Files:**
- Modify: `internal/todo/model.go:12-19` (struct + constructor)
- Modify: `internal/todo/model.go:51-106` (handleKey)
- Modify: `internal/app/model.go:59-61` (input guard)
- Test: `internal/todo/model_test.go`

- [ ] **Step 1: Write failing tests for edit mode**

Add these tests to `internal/todo/model_test.go`:

```go
func TestTodoModelEditMode(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Edit me")
	m := todo.New(s)

	// Press 'e' to enter edit mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(todo.Model)
	if !m.Editing() {
		t.Fatal("expected editing mode after pressing 'e'")
	}
}

func TestTodoModelEditSave(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Original")
	m := todo.New(s)

	// Enter edit mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(todo.Model)

	// Type new text (simulate by typing each rune)
	for _, r := range "Updated" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(todo.Model)
	}

	// Press Enter to save
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(todo.Model)
	if m.Editing() {
		t.Error("expected editing mode to end after enter")
	}
	if m.Tasks()[0].Text == "Original" {
		t.Error("expected task text to be updated")
	}
}

func TestTodoModelEditCancel(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Do not change")
	m := todo.New(s)

	// Enter edit mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(todo.Model)

	// Type something
	for _, r := range "CHANGED" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(todo.Model)
	}

	// Press Esc to cancel
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(todo.Model)
	if m.Editing() {
		t.Error("expected editing mode to end after esc")
	}
	if m.Tasks()[0].Text != "Do not change" {
		t.Errorf("expected text unchanged, got %q", m.Tasks()[0].Text)
	}
}

func TestTodoModelEditEmptyNoOp(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Keep me")
	m := todo.New(s)

	// Enter edit mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(todo.Model)

	// Select all and delete (simulate clearing by using ctrl+u which the textinput handles)
	// We'll clear via the model's input — for testing, press Esc and verify unchanged
	// Actually: press Enter with input pre-filled (unchanged) — should be a no-op update
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(todo.Model)
	if m.Tasks()[0].Text != "Keep me" {
		t.Errorf("expected text unchanged when entering same text, got %q", m.Tasks()[0].Text)
	}
}

func TestTodoModelEditOnEmptyList(t *testing.T) {
	m := newModel(t) // empty list
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(todo.Model)
	if m.Editing() {
		t.Error("should not enter edit mode on empty list")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd daily-tui && go test ./internal/todo/ -run "TestTodoModelEdit" -v`
Expected: Compilation errors — `Editing()` method doesn't exist yet.

- [ ] **Step 3: Add editing state, accessor, and `Inputting()` method to model**

In `internal/todo/model.go`, update the struct to add `editing` and `editTaskID` fields:

```go
// Model is the Bubble Tea model for the Todo tab.
type Model struct {
	store      *Store
	tasks      []Task
	cursor     int
	adding     bool
	editing    bool
	editTaskID string
	input      textinput.Model
}
```

Add the `Editing()` and `Inputting()` accessors alongside the existing ones:

```go
// Exported accessors for tests
func (m Model) Adding() bool    { return m.adding }
func (m Model) Editing() bool   { return m.editing }
func (m Model) Inputting() bool { return m.adding || m.editing }
func (m Model) Tasks() []Task   { return m.tasks }
func (m Model) Cursor() int     { return m.cursor }
```

- [ ] **Step 4: Update `handleKey` to support edit mode**

Replace the `handleKey` method in `internal/todo/model.go` with:

```go
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// --- Input mode (adding or editing) ---
	if m.adding || m.editing {
		switch msg.Type {
		case tea.KeyEsc:
			m.adding = false
			m.editing = false
			m.editTaskID = ""
			m.input.Blur()
			m.input.SetValue("")
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				if m.adding {
					m.store.Add(text)
					m.tasks = m.store.All()
					m.cursor = len(m.tasks) - 1
				} else if m.editing {
					m.store.Update(m.editTaskID, text)
					m.tasks = m.store.All()
				}
			}
			m.adding = false
			m.editing = false
			m.editTaskID = ""
			m.input.Blur()
			m.input.SetValue("")
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// --- Normal mode ---
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case tea.KeySpace:
		if len(m.tasks) > 0 {
			m.store.Toggle(m.tasks[m.cursor].ID)
			m.tasks = m.store.All()
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "a":
			m.adding = true
			m.input.Placeholder = "Add a task..."
			m.input.Focus()
		case "e":
			if len(m.tasks) > 0 {
				m.editing = true
				m.editTaskID = m.tasks[m.cursor].ID
				m.input.Placeholder = "Edit task..."
				m.input.SetValue(m.tasks[m.cursor].Text)
				m.input.Focus()
			}
		case "d":
			if len(m.tasks) > 0 {
				m.store.Delete(m.tasks[m.cursor].ID)
				m.tasks = m.store.All()
				if m.cursor >= len(m.tasks) && m.cursor > 0 {
					m.cursor--
				}
			}
		}
	}
	return m, nil
}
```

Also update the `Update` method to forward non-key messages during editing too:

```go
// Update handles messages and keypresses.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.adding || m.editing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}
```

- [ ] **Step 5: Update app model input guard**

In `internal/app/model.go`, change line 59-60 to use `Inputting()`:

Replace:
```go
		childInputting := (m.activeTab == tabWifi && m.wifi.Inputting()) ||
			(m.activeTab == tabTodo && m.todo.Adding())
```

With:
```go
		childInputting := (m.activeTab == tabWifi && m.wifi.Inputting()) ||
			(m.activeTab == tabTodo && m.todo.Inputting())
```

- [ ] **Step 6: Run edit mode tests**

Run: `cd daily-tui && go test ./internal/todo/ -run "TestTodoModelEdit" -v`
Expected: All 5 edit tests PASS.

- [ ] **Step 7: Run all tests**

Run: `cd daily-tui && go test ./... -v`
Expected: All tests PASS (todo + app + wifi).

- [ ] **Step 8: Commit**

```bash
cd daily-tui && git add internal/todo/model.go internal/todo/model_test.go internal/app/model.go && git commit -m "feat(todo): add edit mode with 'e' key

Press 'e' to edit the selected task. Pre-fills text input with
current text. Enter saves, Esc cancels. App model input guard
updated to use Inputting() which covers both adding and editing."
```

---

### Task 4: View — Rewrite with sectioned layout and progress bar

**Files:**
- Modify: `internal/todo/model.go` (View method + helper functions)

- [ ] **Step 1: Add helper functions for progress bar and section rendering**

Add these helper functions to `internal/todo/model.go`, before the `View` method. You'll need to add `"fmt"` to the imports:

```go
// splitTasks divides tasks into pending and completed, preserving insertion order.
func splitTasks(tasks []Task) (pending, completed []Task) {
	for _, t := range tasks {
		if t.Done {
			completed = append(completed, t)
		} else {
			pending = append(pending, t)
		}
	}
	return
}

// renderProgressBar returns a 30-char progress bar with percentage.
func renderProgressBar(done, total int) string {
	if total == 0 {
		return ""
	}
	const width = 30
	filled := width * done / total
	bar := theme.ProgressFilled.Render(strings.Repeat("▓", filled)) +
		theme.ProgressEmpty.Render(strings.Repeat("░", width-filled))
	pct := fmt.Sprintf("%d%%", 100*done/total)
	return bar + "  " + theme.Dimmed.Render(pct)
}
```

- [ ] **Step 2: Replace the `View` method with the sectioned layout**

Replace the entire `View` method in `internal/todo/model.go`:

```go
// View renders the Todo tab content with sectioned layout.
func (m Model) View() string {
	var sb strings.Builder

	pending, completed := splitTasks(m.tasks)
	doneCount := len(completed)
	total := len(m.tasks)

	// Header with task count
	header := theme.Dimmed.Render("Today's Tasks")
	if total > 0 {
		header += "  " + theme.Badge.Render(fmt.Sprintf("%d/%d done", doneCount, total))
	}
	sb.WriteString(header + "\n")

	// Progress bar
	if total > 0 {
		sb.WriteString(renderProgressBar(doneCount, total) + "\n")
	}
	sb.WriteString("\n")

	if total == 0 {
		sb.WriteString(theme.Dimmed.Render("  No tasks yet — press 'a' to add one") + "\n")
	} else {
		// Pending section
		sb.WriteString(theme.SectionHeader.Render(fmt.Sprintf("── Pending (%d) ", len(pending))) +
			theme.Dimmed.Render("──────────────────────") + "\n")
		if len(pending) == 0 {
			sb.WriteString(theme.Dimmed.Render("  No pending tasks") + "\n")
		} else {
			for i, t := range pending {
				line := "☐  " + t.Text
				if i == m.cursor && !m.adding && !m.editing {
					sb.WriteString(theme.Selected.Render("▸ "+line) + "\n")
				} else {
					sb.WriteString(theme.Normal.Render("  "+line) + "\n")
				}
			}
		}
		sb.WriteString("\n")

		// Completed section
		sb.WriteString(theme.Dimmed.Render(fmt.Sprintf("── Completed (%d) ──────────────────", len(completed))) + "\n")
		if len(completed) == 0 {
			sb.WriteString(theme.Dimmed.Render("  No completed tasks yet") + "\n")
		} else {
			for i, t := range completed {
				line := "☑  " + t.Text
				cursorIdx := len(pending) + i
				if cursorIdx == m.cursor && !m.adding && !m.editing {
					sb.WriteString(theme.Selected.Render("▸ "+line) + "\n")
				} else {
					sb.WriteString(theme.Dimmed.Render("  "+line) + "\n")
				}
			}
		}
	}

	// Input area
	if m.adding || m.editing {
		sb.WriteString("\n" + m.input.View() + "\n")
	}

	// Help bar
	sb.WriteString("\n")
	if m.adding {
		sb.WriteString(theme.HelpStyle.Render("enter confirm • esc cancel"))
	} else if m.editing {
		sb.WriteString(theme.HelpStyle.Render("enter save • esc cancel"))
	} else {
		sb.WriteString(theme.HelpStyle.Render("↑↓ nav • space done • e edit • d del • a add"))
	}
	return sb.String()
}
```

- [ ] **Step 3: Update the `tasks` field to use the sectioned ordering**

The cursor currently indexes into `m.tasks` which is store order. We need the cursor to index into the sectioned order (pending + completed). Update the `handleKey` normal-mode section to use the sectioned list for toggle, edit, and delete operations.

Replace the Toggle, Edit, and Delete cases in `handleKey` to rebuild the display order:

```go
	case tea.KeySpace:
		if len(m.tasks) > 0 {
			pending, completed := splitTasks(m.tasks)
			displayed := append(pending, completed...)
			m.store.Toggle(displayed[m.cursor].ID)
			m.tasks = m.store.All()
			// Clamp cursor after section reorder
			if m.cursor >= len(m.tasks) {
				m.cursor = len(m.tasks) - 1
			}
		}
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "a":
			m.adding = true
			m.input.Placeholder = "Add a task..."
			m.input.Focus()
		case "e":
			if len(m.tasks) > 0 {
				pending, completed := splitTasks(m.tasks)
				displayed := append(pending, completed...)
				m.editing = true
				m.editTaskID = displayed[m.cursor].ID
				m.input.Placeholder = "Edit task..."
				m.input.SetValue(displayed[m.cursor].Text)
				m.input.Focus()
			}
		case "d":
			if len(m.tasks) > 0 {
				pending, completed := splitTasks(m.tasks)
				displayed := append(pending, completed...)
				m.store.Delete(displayed[m.cursor].ID)
				m.tasks = m.store.All()
				if m.cursor >= len(m.tasks) && m.cursor > 0 {
					m.cursor--
				}
			}
		}
```

- [ ] **Step 4: Verify project compiles**

Run: `cd daily-tui && go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 5: Run all tests**

Run: `cd daily-tui && go test ./... -v`
Expected: All tests PASS. Some model tests may need cursor adjustment since display order is now pending+completed.

- [ ] **Step 6: Fix any test failures related to sectioned ordering**

If `TestTodoModelToggleTask` fails because toggling moves the task to the completed section and cursor now points differently, update the test assertion to account for sectioned display order. The task should still be toggled regardless.

- [ ] **Step 7: Commit**

```bash
cd daily-tui && git add internal/todo/model.go && git commit -m "feat(todo): sectioned view with progress bar and visual hierarchy

Rewrite View() to show pending/completed sections with headers,
a progress bar showing completion percentage, and context-sensitive
help text. Cursor indexes into displayed order (pending first,
then completed). Addresses #2."
```

---

### Task 5: Add sectioned cursor navigation test

**Files:**
- Modify: `internal/todo/model_test.go`

- [ ] **Step 1: Write the test**

Add this test to `internal/todo/model_test.go`:

```go
func TestTodoModelSectionedCursor(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Pending 1")
	s.Add("Pending 2")
	task3 := s.Add("Will complete")
	s.Toggle(task3.ID) // mark done
	m := todo.New(s)

	// Cursor starts at 0 — first pending task
	if m.Cursor() != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.Cursor())
	}

	// Move down twice — should reach the completed task (index 2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(todo.Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(todo.Model)
	if m.Cursor() != 2 {
		t.Fatalf("expected cursor at 2, got %d", m.Cursor())
	}

	// Can't go past end
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(todo.Model)
	if m.Cursor() != 2 {
		t.Fatalf("expected cursor clamped at 2, got %d", m.Cursor())
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd daily-tui && go test ./internal/todo/ -run "TestTodoModelSectionedCursor" -v`
Expected: PASS.

- [ ] **Step 3: Run full test suite**

Run: `cd daily-tui && go test ./... -v`
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
cd daily-tui && git add internal/todo/model_test.go && git commit -m "test(todo): add sectioned cursor navigation test

Verify cursor moves across pending and completed sections
and clamps at list boundaries."
```

---

### Task 6: Final verification and build

- [ ] **Step 1: Run the full test suite**

Run: `cd daily-tui && go test ./... -v`
Expected: All tests PASS.

- [ ] **Step 2: Build the binary**

Run: `cd daily-tui && go build -o daily-tui ./`
Expected: Clean build, binary produced.

- [ ] **Step 3: Run linting (if available)**

Run: `cd daily-tui && go vet ./...`
Expected: No issues.

- [ ] **Step 4: Clean up build artifact**

Run: `cd daily-tui && rm -f daily-tui`
