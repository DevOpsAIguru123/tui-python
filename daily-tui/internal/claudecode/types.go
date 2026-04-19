package claudecode

// Command is one entry in the Claude Code tab's launcher list.
//
// Prompt is the exact string passed as the argument to `claude -p`. For
// user-authored slash commands discovered in ~/.claude/commands/*.md this
// is "/<name>" (so Claude Code expands the template server-side); for the
// built-in entries it's a plain prompt like "hello".
//
// ExtraArgs are appended to the argv after the prompt — use them sparingly
// for commands that legitimately need non-default flags (e.g. a skill that
// needs --dangerously-skip-permissions because it shells out to system
// tools). They render inline on the command row so the user can see what
// they're consenting to before pressing Enter.
type Command struct {
	Name      string   // display label, e.g. "hello" or "network-monitor"
	Prompt    string   // verbatim argument to `claude -p`
	ExtraArgs []string // optional extra flags appended to the claude invocation
	Source    string   // "builtin", "user", or "project" — shown as a tag in the row
	Path      string   // file path, empty for builtins
}
