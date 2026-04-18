package claudecode

import (
	"context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// RunDoneMsg is delivered to the model once `claude -p` exits (successfully
// or not). Name identifies which command in the launcher list this result
// belongs to — needed because multiple commands can be running concurrently
// and we have to route each completion to its own row. Output contains
// combined stdout/stderr; Err is the exec error (including context-timeout
// if the 5-minute cap was hit).
type RunDoneMsg struct {
	Name   string
	Prompt string
	Output string
	Err    error
}

// runTimeout bounds a single invocation. Long enough for a realistic Claude
// response, short enough that a hung process can't silently block the tab
// forever.
const runTimeout = 5 * time.Minute

// RunCommandCmd shells out to `claude -p <prompt> [extraArgs...]` and returns
// a RunDoneMsg when it finishes. CombinedOutput is used so any error text
// printed on stderr (auth failure, invalid flag, etc.) still reaches the
// user — a silently-empty pane is worse than a visible error.
//
// name is echoed back in the resulting RunDoneMsg so the caller can route
// the completion to the right launcher row when several runs are in flight
// at once. extraArgs are appended after the prompt — they exist so a
// command entry can carry extra flags (for example
// --dangerously-skip-permissions) that a specific skill invocation requires.
func RunCommandCmd(name, prompt string, extraArgs ...string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
		defer cancel()
		argv := append([]string{"-p", prompt}, extraArgs...)
		out, err := exec.CommandContext(ctx, "claude", argv...).CombinedOutput()
		return RunDoneMsg{Name: name, Prompt: prompt, Output: string(out), Err: err}
	}
}
