package claudecode

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"
)

// Discover returns the list of commands available in the Claude Code tab.
//
// Ordering (stable across runs):
//  1. builtins (hardcoded below)
//  2. config entries from ~/.config/daily-tui/config.yaml
//  3. user commands from ~/.claude/commands/*.md
//  4. project commands from $PWD/.claude/commands/*.md
//
// Config is accepted as a value (not a pointer) so callers can pass a zero
// value or a nil-safe accessor without panicking. Dirs are each sorted by
// name internally.
func Discover(cfg config.ClaudeCodeConfig) []Command {
	out := []Command{
		{Name: "hello", Prompt: "hello", Source: "builtin"},
		{
			Name:      "network-monitor",
			Prompt:    "run this skill /network-monitor and store results in llm wiki",
			ExtraArgs: []string{"--dangerously-skip-permissions"},
			Source:    "builtin",
		},
	}

	out = append(out, fromConfig(cfg)...)

	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, scanCommandsDir(filepath.Join(home, ".claude", "commands"), "user")...)
	}
	if wd, err := os.Getwd(); err == nil {
		out = append(out, scanCommandsDir(filepath.Join(wd, ".claude", "commands"), "project")...)
	}
	return out
}

// fromConfig turns each config entry into a claudecode.Command, tagging the
// source as "config" so users can spot config-driven entries in the list.
// Entries with an empty Name or Prompt are skipped — we'd rather drop a
// malformed row than surface "<blank>" in the UI.
func fromConfig(cfg config.ClaudeCodeConfig) []Command {
	if len(cfg.Commands) == 0 {
		return nil
	}
	out := make([]Command, 0, len(cfg.Commands))
	for _, c := range cfg.Commands {
		name := strings.TrimSpace(c.Name)
		prompt := strings.TrimSpace(c.Prompt)
		if name == "" || prompt == "" {
			continue
		}
		out = append(out, Command{
			Name:      name,
			Prompt:    prompt,
			ExtraArgs: append([]string(nil), c.ExtraArgs...),
			Source:    "config",
		})
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
