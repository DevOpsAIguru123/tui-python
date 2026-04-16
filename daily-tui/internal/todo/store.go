package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Task represents a single todo item.
type Task struct {
	ID        string     `json:"id"`
	Text      string     `json:"text"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// Store manages todo persistence to a JSON file.
type Store struct {
	path  string
	tasks []Task
}

// NewStore loads tasks from path (creates file if absent).
func NewStore(path string) *Store {
	s := &Store{path: path}
	s.load()
	return s
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &s.tasks)
}

func (s *Store) save() {
	data, err := json.MarshalIndent(s.tasks, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	_ = os.WriteFile(s.path, data, 0644)
}

// All returns a copy of all tasks.
func (s *Store) All() []Task {
	out := make([]Task, len(s.tasks))
	copy(out, s.tasks)
	return out
}

// Add creates a new task and saves immediately.
func (s *Store) Add(text string) Task {
	t := Task{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		Done:      false,
		CreatedAt: time.Now(),
	}
	s.tasks = append(s.tasks, t)
	s.save()
	return t
}

// Toggle flips the Done state of the task with the given ID.
func (s *Store) Toggle(id string) {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Done = !s.tasks[i].Done
			s.save()
			return
		}
	}
}

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

// Delete removes the task with the given ID.
func (s *Store) Delete(id string) {
	filtered := s.tasks[:0]
	for _, t := range s.tasks {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}
	s.tasks = filtered
	s.save()
}
