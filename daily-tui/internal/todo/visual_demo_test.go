package todo_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"

	tea "github.com/charmbracelet/bubbletea"
)

// TestVisualSmoke renders the Todo View at several states and prints the output
// so a human can eyeball the rendering. Run with `go test -run TestVisualSmoke -v`.
func TestVisualSmoke(t *testing.T) {
	print := func(label, s string) {
		fmt.Printf("\n========== %s ==========\n%s\n", label, s)
	}

	// 1. Empty list
	m1 := todo.New(todo.NewStore(filepath.Join(t.TempDir(), "a.json")))
	print("State 1: empty list", m1.View())

	// 2. Mixed tasks (2 pending, 3 done)
	s2 := todo.NewStore(filepath.Join(t.TempDir(), "b.json"))
	s2.Add("Fix login bug")
	s2.Add("Write tests")
	done1 := s2.Add("Set up CI")
	done2 := s2.Add("Add README")
	done3 := s2.Add("Deploy v1")
	s2.Toggle(done1.ID)
	s2.Toggle(done2.ID)
	s2.Toggle(done3.ID)
	m2 := todo.New(s2)
	print("State 2: 2 pending / 3 done, cursor at first pending", m2.View())

	// 3. Cursor moved to second pending
	updated, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
	m3 := updated.(todo.Model)
	print("State 3: cursor on 2nd pending task", m3.View())

	// 4. Cursor in completed section
	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyDown})
	m4 := updated.(todo.Model)
	print("State 4: cursor in completed section", m4.View())

	// 5. Edit mode — press 'e' on first pending task
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m5 := updated.(todo.Model)
	print("State 5: editing first pending task", m5.View())

	// 6. Add mode
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m6 := updated.(todo.Model)
	print("State 6: add mode", m6.View())

	// 7. All pending, no completed yet
	s7 := todo.NewStore(filepath.Join(t.TempDir(), "c.json"))
	s7.Add("Buy coffee")
	s7.Add("Ship PR")
	m7 := todo.New(s7)
	print("State 7: all pending, 0% progress", m7.View())

	// 8. All done
	s8 := todo.NewStore(filepath.Join(t.TempDir(), "d.json"))
	t1 := s8.Add("Done A")
	t2 := s8.Add("Done B")
	s8.Toggle(t1.ID)
	s8.Toggle(t2.ID)
	m8 := todo.New(s8)
	print("State 8: all done, 100% progress", m8.View())

	// Minimal assertions so the test is a real test, not just a printer
	if !strings.Contains(m2.View(), "Pending") || !strings.Contains(m2.View(), "Completed") {
		t.Error("expected both Pending and Completed section headers in mixed view")
	}
	// The "N/M done" label moved to the breadcrumb (Title()). The tab
	// body now shows just the progress bar + percentage.
	if !strings.Contains(m2.View(), "60%") {
		t.Error("expected '60%' progress label in mixed view (3 done of 5)")
	}
	if !strings.Contains(m2.Title(), "2 pending · 3 done") {
		t.Error("expected breadcrumb-ready title '2 pending · 3 done'")
	}
	v5 := m5.View()
	if !strings.Contains(v5, "enter") || !strings.Contains(v5, "save") {
		t.Error("expected edit-mode help keycap 'enter' + 'save' label in edit view")
	}
}
