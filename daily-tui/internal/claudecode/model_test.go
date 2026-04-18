package claudecode_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/claudecode"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewLoadsBuiltinHello(t *testing.T) {
	m := claudecode.New()
	cmds := m.Commands()
	if len(cmds) == 0 {
		t.Fatal("expected at least the builtin 'hello' command")
	}
	if cmds[0].Name != "hello" || cmds[0].Prompt != "hello" {
		t.Errorf("expected first command to be builtin hello, got %+v", cmds[0])
	}
	if cmds[0].Source != "builtin" {
		t.Errorf("expected builtin source tag, got %q", cmds[0].Source)
	}
}

func TestNetworkMonitorBuiltinHasSkipPermissionsFlag(t *testing.T) {
	m := claudecode.New()
	var got *claudecode.Command
	for i := range m.Commands() {
		c := m.Commands()[i]
		if c.Name == "network-monitor" {
			got = &c
			break
		}
	}
	if got == nil {
		t.Fatal("expected 'network-monitor' builtin command to be registered")
	}
	if got.Prompt != "run this skill /network-monitor" {
		t.Errorf("unexpected prompt: %q", got.Prompt)
	}
	if len(got.ExtraArgs) != 1 || got.ExtraArgs[0] != "--dangerously-skip-permissions" {
		t.Errorf("expected ExtraArgs=[--dangerously-skip-permissions], got %v", got.ExtraArgs)
	}
}

func TestCursorNavigation(t *testing.T) {
	m := claudecode.New()
	if m.Cursor() != 0 {
		t.Errorf("expected cursor at 0, got %d", m.Cursor())
	}
	if len(m.Commands()) < 2 {
		t.Skip("not enough commands on the host to test navigation")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(claudecode.Model)
	if m2.Cursor() != 1 {
		t.Errorf("expected cursor at 1 after Down, got %d", m2.Cursor())
	}
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyUp})
	m3 := updated.(claudecode.Model)
	if m3.Cursor() != 0 {
		t.Errorf("expected cursor back at 0 after Up, got %d", m3.Cursor())
	}
}

func TestEnterStartsRun(t *testing.T) {
	m := claudecode.New()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(claudecode.Model)
	if !m2.Inputting() {
		t.Error("expected Inputting()=true after Enter (running state)")
	}
	if cmd == nil {
		t.Error("expected a tea.Cmd (RunCommandCmd) after Enter")
	}
}

func TestRunDoneTransitionsToDone(t *testing.T) {
	m := claudecode.New()
	// Start a run so selected is populated.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Prompt: "hello", Output: "hi there"})
	m2 := updated.(claudecode.Model)
	if m2.Inputting() {
		t.Error("expected not-inputting after RunDoneMsg")
	}
	if !strings.Contains(m2.View(), "hi there") {
		t.Errorf("expected output 'hi there' rendered in View, got:\n%s", m2.View())
	}
}

func TestEscAfterDoneReturnsToBrowse(t *testing.T) {
	m := claudecode.New()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Output: "x"})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := updated.(claudecode.Model)
	if m2.Output() != "" {
		t.Errorf("expected output cleared after esc, got %q", m2.Output())
	}
}

func TestDiscoverSkipsMissingDir(t *testing.T) {
	// Sanity: Discover shouldn't panic or error out when the user has no
	// ~/.claude/commands directory. The builtin must always be present.
	m := claudecode.New()
	got := m.Commands()
	if len(got) < 1 {
		t.Error("expected at least builtin hello, got none")
	}
	// Make sure the builtin's Path is empty (so tests don't depend on any
	// particular host filesystem layout).
	if got[0].Path != "" {
		t.Errorf("expected builtin Path to be empty, got %q", got[0].Path)
	}
	// The test binary's CWD shouldn't pollute results — assert we don't pick
	// up any bogus path entries by checking all Paths are absolute when set.
	for _, c := range got {
		if c.Path != "" && !filepath.IsAbs(c.Path) {
			t.Errorf("command %q has non-absolute path: %q", c.Name, c.Path)
		}
	}
}
