package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/app"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/calendar"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/todo"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("daily-tui " + version)
			return
		case "--help", "-h":
			fmt.Println("daily-tui — terminal dashboard for WiFi & tasks")
			fmt.Println()
			fmt.Println("Usage: daily-tui [flags]")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  -v, --version   Print version and exit")
			fmt.Println("  -h, --help      Show this help message")
			return
		}
	}

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
	calendarModel := calendar.New()
	root := app.New(wifiModel, todoModel, calendarModel, version)

	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
