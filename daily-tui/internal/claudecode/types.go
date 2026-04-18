package claudecode

// Command is one entry in the Claude Code tab's launcher list.
//
// Prompt is the exact string passed as the argument to `claude -p`. For
// user-authored slash commands discovered in ~/.claude/commands/*.md this
// is "/<name>" (so Claude Code expands the template server-side); for the
// built-in entries it's a plain prompt like "hello".
type Command struct {
	Name   string // display label, e.g. "hello" or "review"
	Prompt string // verbatim argument to `claude -p`
	Source string // "builtin", "user", or "project" — shown as a tag in the row
	Path   string // file path, empty for builtins
}
