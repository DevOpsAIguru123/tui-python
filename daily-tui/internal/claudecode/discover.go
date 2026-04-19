package claudecode

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Discover returns the list of commands available in the Claude Code tab.
// The ordering is stable: builtins first, then user commands from
// ~/.claude/commands, then project commands from $PWD/.claude/commands.
// Each group is sorted by name so the list is deterministic across runs.
func Discover() []Command {
	out := []Command{
		{Name: "hello", Prompt: "hello", Source: "builtin"},
		{
			Name:      "network-monitor",
			Prompt:    "run this skill /network-monitor and store results in llm wiki",
			ExtraArgs: []string{"--dangerously-skip-permissions"},
			Source:    "builtin",
		},
	}

	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, scanCommandsDir(filepath.Join(home, ".claude", "commands"), "user")...)
	}
	if wd, err := os.Getwd(); err == nil {
		out = append(out, scanCommandsDir(filepath.Join(wd, ".claude", "commands"), "project")...)
	}
	return out
}

// scanCommandsDir walks a ".claude/commands" tree and turns every *.md file
// into a Command. Nested directories become colon-namespaces in the command
// name, mirroring Claude Code's own convention (e.g. utils/foo.md → utils:foo
// on disk, "/utils:foo" as the slash command to invoke).
func scanCommandsDir(root, source string) []Command {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	var cmds []Command
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		name := strings.TrimSuffix(rel, ".md")
		name = strings.ReplaceAll(name, string(filepath.Separator), ":")
		cmds = append(cmds, Command{
			Name:   name,
			Prompt: "/" + name,
			Source: source,
			Path:   path,
		})
		return nil
	})
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })
	return cmds
}
