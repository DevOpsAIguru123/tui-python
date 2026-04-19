package claudecode_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/claudecode"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewLoadsBuiltinHello(t *testing.T) {
	m := claudecode.New(config.ClaudeCodeConfig{})
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

func TestConfigCommandsAppearInList(t *testing.T) {
	cfg := config.ClaudeCodeConfig{
		Commands: []config.ClaudeCodeCommand{
			{Name: "deploy", Prompt: "run the deploy pipeline"},
			{Name: "security-scan", Prompt: "run /security-review",
				ExtraArgs: []string{"--dangerously-skip-permissions"}},
			// Entries with blank Name/Prompt must be dropped silently.
			{Name: "", Prompt: "ignored"},
			{Name: "ignored-no-prompt", Prompt: ""},
		},
	}
	m := claudecode.New(cfg)

	findByName := func(want string) *claudecode.Command {
		for i := range m.Commands() {
			c := m.Commands()[i]
			if c.Name == want {
				return &c
			}
		}
		return nil
	}

	deploy := findByName("deploy")
	if deploy == nil {
		t.Fatal("expected config-driven 'deploy' command to be discovered")
	}
	if deploy.Source != "config" {
		t.Errorf("expected Source=\"config\" for config entry, got %q", deploy.Source)
	}
	if deploy.Prompt != "run the deploy pipeline" {
		t.Errorf("unexpected Prompt: %q", deploy.Prompt)
	}

	scan := findByName("security-scan")
	if scan == nil {
		t.Fatal("expected config-driven 'security-scan' command to be discovered")
	}
	if len(scan.ExtraArgs) != 1 || scan.ExtraArgs[0] != "--dangerously-skip-permissions" {
		t.Errorf("expected ExtraArgs propagated from config, got %v", scan.ExtraArgs)
	}

	if findByName("") != nil || findByName("ignored-no-prompt") != nil {
		t.Error("expected entries with blank Name or Prompt to be dropped")
	}

	// Ordering: builtins come before config entries.
	for _, c := range m.Commands() {
		if c.Source == "config" {
			break // reached config section without hitting builtins first? we check below
		}
		if c.Source != "builtin" {
			t.Errorf("expected builtins before any other source, saw %q (name=%s)", c.Source, c.Name)
			break
		}
	}
}

func TestNetworkMonitorBuiltinHasSkipPermissionsFlag(t *testing.T) {
	m := claudecode.New(config.ClaudeCodeConfig{})
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
	m := claudecode.New(config.ClaudeCodeConfig{})
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
	m := claudecode.New(config.ClaudeCodeConfig{})
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(claudecode.Model)
	if cmd == nil {
		t.Error("expected a tea.Cmd (RunCommandCmd) after Enter")
	}
	// The cursor's command should now have a runState recorded.
	first := m2.Commands()[0].Name
	if r := m2.Runs()[first]; r == nil {
		t.Errorf("expected runs[%q] to be created after Enter", first)
	}
	// Inputting must remain false even during a run so the app chrome
	// can still intercept tab/number keys for navigation.
	if m2.Inputting() {
		t.Error("expected Inputting()=false during runs so tab-switching still works")
	}
}

func TestRunDoneRendersOutput(t *testing.T) {
	m := claudecode.New(config.ClaudeCodeConfig{})
	// The viewport needs a size before it will render its content — the app
	// provides this on WindowSizeMsg. Do the equivalent here.
	m.SetRowWidth(80)
	m.SetContentHeight(30)
	first := m.Commands()[0].Name
	// Start a run and deliver a RunDoneMsg matching that command.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Name: first, Prompt: "hello", Output: "hi there"})
	m2 := updated.(claudecode.Model)
	if m2.Output() != "hi there" {
		t.Errorf("expected Output()='hi there', got %q", m2.Output())
	}
	if !strings.Contains(m2.View(), "hi there") {
		t.Errorf("expected output 'hi there' rendered in View, got:\n%s", m2.View())
	}
}

func TestArrowKeysMoveCursorAfterRun(t *testing.T) {
	// Regression: after a command finishes, ↓ used to be eaten by the output
	// viewport, leaving the user stuck on whichever command they'd just
	// run. Now arrows always navigate the command list — the user can
	// move from "hello" to "network-monitor" and Enter to fire it.
	m := claudecode.New(config.ClaudeCodeConfig{})
	m.SetRowWidth(80)
	m.SetContentHeight(30)
	if len(m.Commands()) < 2 {
		t.Skip("need at least two commands to test cursor movement")
	}
	first := m.Commands()[0].Name
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Name: first, Output: "x"})
	m = updated.(claudecode.Model)
	if m.Cursor() != 0 {
		t.Fatalf("setup: expected cursor at 0 after run, got %d", m.Cursor())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(claudecode.Model)
	if m2.Cursor() != 1 {
		t.Errorf("expected ↓ to advance cursor to 1, got %d", m2.Cursor())
	}
	updated, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := updated.(claudecode.Model)
	if cmd == nil {
		t.Error("expected Enter on new cursor position to fire a tea.Cmd")
	}
	second := m3.Commands()[1].Name
	if r := m3.Runs()[second]; r == nil {
		t.Errorf("expected runs[%q] to be created", second)
	}
}

func TestConcurrentRuns(t *testing.T) {
	// Two commands fired back-to-back must each get their own runState
	// and complete independently — the second Enter press shouldn't
	// cancel or replace the first run.
	m := claudecode.New(config.ClaudeCodeConfig{})
	m.SetRowWidth(80)
	m.SetContentHeight(30)
	if len(m.Commands()) < 2 {
		t.Skip("need at least two commands to test concurrent runs")
	}
	first := m.Commands()[0].Name
	second := m.Commands()[1].Name

	// Fire #1.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	// Move cursor and fire #2 while #1 is still "running" (no RunDoneMsg yet).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(claudecode.Model)
	updated, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	if cmd2 == nil {
		t.Error("expected second Enter to fire its own tea.Cmd while first run is still in flight")
	}
	if r := m.Runs()[first]; r == nil {
		t.Errorf("expected first command's runState preserved while second runs; runs[%q] is nil", first)
	}
	if r := m.Runs()[second]; r == nil {
		t.Errorf("expected second command's runState created; runs[%q] is nil", second)
	}

	// Deliver completion of the SECOND command first (out-of-order).
	updated, _ = m.Update(claudecode.RunDoneMsg{Name: second, Output: "two"})
	m = updated.(claudecode.Model)
	// Then the first.
	updated, _ = m.Update(claudecode.RunDoneMsg{Name: first, Output: "one"})
	m = updated.(claudecode.Model)

	if got := m.Runs()[first].Output; got != "one" {
		t.Errorf("expected runs[%q].Output='one', got %q", first, got)
	}
	if got := m.Runs()[second].Output; got != "two" {
		t.Errorf("expected runs[%q].Output='two', got %q", second, got)
	}
}

func TestDuplicateEnterIgnoredWhileRunning(t *testing.T) {
	// Pressing Enter twice in a row on the same command shouldn't stack
	// runs — the second press should be a no-op until the first completes.
	m := claudecode.New(config.ClaudeCodeConfig{})
	first := m.Commands()[0].Name
	updated, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	if cmd1 == nil {
		t.Fatal("expected first Enter to produce a tea.Cmd")
	}
	updated, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = updated
	if cmd2 != nil {
		t.Errorf("expected second Enter on still-running %q to be ignored, got cmd %v", first, cmd2)
	}
}

func TestEscClearsFocusedDoneRun(t *testing.T) {
	m := claudecode.New(config.ClaudeCodeConfig{})
	first := m.Commands()[0].Name
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(claudecode.RunDoneMsg{Name: first, Output: "x"})
	m = updated.(claudecode.Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := updated.(claudecode.Model)
	if m2.Output() != "" {
		t.Errorf("expected output cleared after esc, got %q", m2.Output())
	}
	if _, ok := m2.Runs()[first]; ok {
		t.Errorf("expected runs[%q] removed after esc, but it's still there", first)
	}
}

func TestDiscoverSkipsMissingDir(t *testing.T) {
	// Sanity: Discover shouldn't panic or error out when the user has no
	// ~/.claude/commands directory. The builtin must always be present.
	m := claudecode.New(config.ClaudeCodeConfig{})
	got := m.Commands()
	if len(got) < 1 {
		t.Error("expected at least builtin hello, got none")
	}
	if got[0].Path != "" {
		t.Errorf("expected builtin Path to be empty, got %q", got[0].Path)
	}
	for _, c := range got {
		if c.Path != "" && !filepath.IsAbs(c.Path) {
			t.Errorf("command %q has non-absolute path: %q", c.Name, c.Path)
		}
	}
}
