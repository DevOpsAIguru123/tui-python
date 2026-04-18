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
	if got.Prompt != "run this skill /network-monitor and store results in llm wiki" {
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
	if m2.State() == 0 {
		t.Error("expected state to advance past browsing after Enter")
	}
	if cmd == nil {
		t.Error("expected a tea.Cmd (RunCommandCmd) after Enter")
	}
	// Inputting must remain false even during a run so the app chrome
	// can still intercept tab/number keys for navigation.
	if m2.Inputting() {
		t.Error("expected Inputting()=false during runs so tab-switching still works")
	}
}

func TestRunDoneTransitionsToDone(t *testing.T) {
	m := claudecode.New()
	// The viewport needs a size before it will render its content — the app
	// provides this on WindowSizeMsg. Do the equivalent here.
	m.SetRowWidth(80)
	m.SetContentHeight(30)
	// Start a run so selected is populated.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Prompt: "hello", Output: "hi there"})
	m2 := updated.(claudecode.Model)
	if m2.Output() != "hi there" {
		t.Errorf("expected Output()='hi there', got %q", m2.Output())
	}
	if !strings.Contains(m2.View(), "hi there") {
		t.Errorf("expected output 'hi there' rendered in View, got:\n%s", m2.View())
	}
}

func TestArrowKeysMoveCursorInDoneState(t *testing.T) {
	// Regression: in the done state, ↑↓ used to be routed to the output
	// viewport, leaving the user stuck on whichever command they'd just run
	// — they couldn't move the cursor to "network-monitor" and press Enter
	// to run it. Now arrows always navigate the command list.
	m := claudecode.New()
	m.SetRowWidth(80)
	m.SetContentHeight(30)
	if len(m.Commands()) < 2 {
		t.Skip("need at least two commands to test cursor movement")
	}
	// Run the first command.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Prompt: "hello", Output: "x"})
	m = updated.(claudecode.Model)
	if m.Cursor() != 0 {
		t.Fatalf("setup: expected cursor at 0 after run, got %d", m.Cursor())
	}
	// In done state, ↓ must advance the command-list cursor.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(claudecode.Model)
	if m2.Cursor() != 1 {
		t.Errorf("expected ↓ in done state to move cursor to 1, got %d", m2.Cursor())
	}
	// Pressing Enter on the new cursor position should fire that command.
	updated, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := updated.(claudecode.Model)
	if cmd == nil {
		t.Error("expected Enter on new cursor position to fire a tea.Cmd")
	}
	if m3.State() == 0 { // stateBrowsing
		t.Error("expected state to advance past browsing after Enter on new cursor")
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
