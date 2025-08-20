package tui

import (
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	tea "github.com/charmbracelet/bubbletea"
	"reapo/internal/tools"
)

// RunTUI starts the TUI interface
func RunTUI(client anthropic.Client, toolDefs []tools.ToolDefinition, sysMessages []anthropic.TextBlockParam) {
	// Set the system messages for the TUI package
	systemMessages = sysMessages

	// Create the TUI model
	m := NewModel(client, toolDefs)

	// Run the Bubble Tea program with mouse support
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}
