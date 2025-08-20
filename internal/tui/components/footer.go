package components

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"reapo/internal/config"
	"reapo/internal/tui/components/vimtextarea"
)

// FooterComponent handles the rendering of the status bar footer
type FooterComponent struct {
	mode             vimtextarea.Mode
	width            int
	contextTokens    int
	maxContextTokens int
	modelName        string
	focusedWindow    string // "chat" or "input"
	chatCursorLine   int    // Current cursor line in chat
	chatTotalLines   int    // Total lines in chat
}

// NewFooterComponent creates a new footer component
func NewFooterComponent(mode vimtextarea.Mode, width int) *FooterComponent {
	return &FooterComponent{
		mode:             mode,
		width:            width,
		contextTokens:    0,
		maxContextTokens: config.GetContextTokens(),
		modelName:        config.GetModelName(),
		focusedWindow:    "input",
	}
}

// SetFocusedWindow updates which window is focused
func (f *FooterComponent) SetFocusedWindow(window string) {
	f.focusedWindow = window
}

// SetChatCursorInfo updates the chat cursor position information
func (f *FooterComponent) SetChatCursorInfo(cursorLine, totalLines int) {
	f.chatCursorLine = cursorLine
	f.chatTotalLines = totalLines
}

// Render renders the complete footer with mode indicator and status bar
func (f *FooterComponent) Render() string {
	// Create mode indicator based on focused window
	var modeIndicator *ModeIndicatorComponent
	var modeIndicatorRendered string
	var modeIndicatorWidth int
	
	if f.focusedWindow == "chat" {
		// Show CHAT mode when chat is focused
		chatModeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("6")). // Cyan for chat
			Foreground(lipgloss.Color("0"))
		modeIndicatorRendered = chatModeStyle.Render("  CHAT  ")
		modeIndicatorWidth = 8
	} else {
		// Show vim mode when input is focused
		modeIndicator = NewModeIndicatorComponent(f.mode)
		modeIndicatorRendered = modeIndicator.Render()
		modeIndicatorWidth = modeIndicator.Width()
	}

	// Calculate remaining width for main footer content
	remainingWidth := f.width - modeIndicatorWidth

	// Get current directory info
	pwd, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" && strings.HasPrefix(pwd, homeDir) {
		pwd = "~" + pwd[len(homeDir):]
	}

	// Format context usage
	percentage := float64(f.contextTokens) / float64(f.maxContextTokens) * 100
	contextText := fmt.Sprintf("%s/%s (%.1f%%)",
		formatTokenCount(f.contextTokens),
		formatTokenCount(f.maxContextTokens),
		percentage)

	// Choose color based on usage percentage
	var contextColor string
	if percentage < 50 {
		contextColor = "2" // Green
	} else if percentage < 80 {
		contextColor = "3" // Yellow
	} else {
		contextColor = "1" // Red
	}

	leftText := "reapo"
	rightText := f.modelName
	
	// Add cursor position if in chat mode
	if f.focusedWindow == "chat" && f.chatTotalLines > 0 {
		// Show actual line position in the chat (1-based for user)
		cursorPosition := fmt.Sprintf("Line %d/%d", f.chatCursorLine+1, f.chatTotalLines)
		rightText = cursorPosition + " | " + rightText
	}

	// Build the sections with proper spacing
	// Layout: reapo | pwd | context | [cursor pos |] model
	sections := []string{leftText, pwd, contextText, rightText}

	// Calculate spacing between sections
	totalContentWidth := 0
	for _, section := range sections {
		totalContentWidth += len(section)
	}

	// Account for separators (3 spaces between each section) and padding
	separatorCount := len(sections) - 1
	totalSeparatorWidth := separatorCount * 3
	availableWidth := remainingWidth - totalContentWidth - totalSeparatorWidth - 2

	// Distribute extra space evenly
	extraSpacePerGap := availableWidth / separatorCount
	if extraSpacePerGap < 0 {
		extraSpacePerGap = 0
	}

	// Create the colored context text
	contextStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(contextColor)).
		Background(lipgloss.Color("236")).
		Render(contextText)

	// Build footer parts
	separator := strings.Repeat(" ", 3+extraSpacePerGap)

	// Style each part with consistent background
	styledLeft := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("236")).
		Render(leftText)

	styledPwd := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("236")).
		Render(pwd)

	styledRight := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Background(lipgloss.Color("236")).
		Render(rightText)

	styledSeparator := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Render(separator)

	// Compose the footer
	composedFooter := styledLeft + styledSeparator + styledPwd + styledSeparator + contextStyled + styledSeparator + styledRight

	// Ensure the footer fills the entire width with padding
	paddingNeeded := remainingWidth - lipgloss.Width(composedFooter) - 2 // -2 for left/right padding
	if paddingNeeded > 0 {
		composedFooter += lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Render(strings.Repeat(" ", paddingNeeded))
	}

	mainFooter := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(remainingWidth).
		Padding(0, 1).
		Render(composedFooter)

	// Combine mode indicator and main footer
	return modeIndicatorRendered + mainFooter
}

// UpdateContextInfo updates the context token information
func (f *FooterComponent) UpdateContextInfo(contextTokens, maxContextTokens int, modelName string) {
	f.contextTokens = contextTokens
	f.maxContextTokens = maxContextTokens
	f.modelName = modelName
}

// formatTokenCount formats token count with k suffix for thousands
func formatTokenCount(tokens int) string {
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}
