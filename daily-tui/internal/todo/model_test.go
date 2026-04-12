package todo_test

import (
	"path/filepath"
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"

	tea "github.com/charmbracelet/bubbletea"
)

func newModel(t *testing.T) todo.Model {
	t.Helper()
	return todo.New(todo.NewStore(filepath.Join(t.TempDir(), "todos.json")))
}

func TestTodoModelAddMode(t *testing.T) {
	m := newModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(todo.Model)
	if !m.Adding() {
		t.Fatal("expected adding mode after pressing 'a'")
	}
}

func TestTodoModelEscCancelsAdd(t *testing.T) {
	m := newModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(todo.Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(todo.Model)
	if m.Adding() {
		t.Error("expected add mode cancelled after esc")
	}
}

func TestTodoModelDeleteTask(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Task to delete")
	m := todo.New(s)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(todo.Model)
	if len(m.Tasks()) != 0 {
		t.Errorf("expected 0 tasks after delete, got %d", len(m.Tasks()))
	}
}

func TestTodoModelToggleTask(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	s.Add("Toggle me")
	m := todo.New(s)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(todo.Model)
	if !m.Tasks()[0].Done {
		t.Error("expected task done after space")
	}
}
