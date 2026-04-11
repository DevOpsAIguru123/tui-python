package main

import (
	"fmt"
	"os"
	"path/filepath"

	"daily-tui/internal/app"
	"daily-tui/internal/config"
	"daily-tui/internal/todo"
	"daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "daily-tui")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating config dir: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	wifiModel := wifi.New(cfg)
	todoStore := todo.NewStore(filepath.Join(configDir, "todos.json"))
	todoModel := todo.New(todoStore)
	root := app.New(wifiModel, todoModel)

	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
