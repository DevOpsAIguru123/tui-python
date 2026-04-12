package todo_test

import (
	"path/filepath"
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
)

func TestStoreAddTask(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	task := s.Add("Buy coffee")
	if task.Text != "Buy coffee" {
		t.Errorf("expected 'Buy coffee', got %q", task.Text)
	}
	if task.Done {
		t.Error("new task should not be done")
	}
	if task.ID == "" {
		t.Error("task ID should not be empty")
	}
}

func TestStoreToggleTask(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	task := s.Add("Morning standup")
	s.Toggle(task.ID)
	if !s.All()[0].Done {
		t.Error("expected task done after toggle")
	}
	s.Toggle(task.ID)
	if s.All()[0].Done {
		t.Error("expected task not done after second toggle")
	}
}

func TestStoreDeleteTask(t *testing.T) {
	s := todo.NewStore(filepath.Join(t.TempDir(), "todos.json"))
	task := s.Add("Delete me")
	s.Delete(task.ID)
	if len(s.All()) != 0 {
		t.Error("expected 0 tasks after delete")
	}
}

func TestStorePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")
	s1 := todo.NewStore(path)
	s1.Add("Persistent task")

	s2 := todo.NewStore(path)
	tasks := s2.All()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task after reload, got %d", len(tasks))
	}
	if tasks[0].Text != "Persistent task" {
		t.Errorf("expected 'Persistent task', got %q", tasks[0].Text)
	}
}
