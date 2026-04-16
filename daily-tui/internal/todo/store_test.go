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
