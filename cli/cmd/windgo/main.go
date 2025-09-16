package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wilfierd/windgo-chat-app/cli/internal/api"
	"github.com/wilfierd/windgo-chat-app/cli/internal/ui"
)

func main() {
	client := api.NewClient()
	program := tea.NewProgram(ui.NewModel(client), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
