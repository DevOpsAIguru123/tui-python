package claudecode

import (
	"context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// RunDoneMsg is delivered to the model once `claude -p` exits (successfully
// or not). Output contains combined stdout/stderr; Err is the exec error
// (including context-timeout if the 5-minute cap was hit).
type RunDoneMsg struct {
	Prompt string
	Output string
	Err    error
}

// runTimeout bounds a single invocation. Long enough for a realistic Claude
// response, short enough that a hung process can't silently block the tab
// forever.
const runTimeout = 5 * time.Minute

// RunCommandCmd shells out to `claude -p <prompt>` and returns a RunDoneMsg
// when it finishes. CombinedOutput is used so any error text printed on
// stderr (auth failure, invalid flag, etc.) still reaches the user — a
// silently-empty pane is worse than a visible error.
func RunCommandCmd(prompt string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "claude", "-p", prompt).CombinedOutput()
		return RunDoneMsg{Prompt: prompt, Output: string(out), Err: err}
	}
}
