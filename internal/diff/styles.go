package diff

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Define reusable styles for diff formatting
var (
	// Line change styles
	DeletedLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("52")).
				Foreground(lipgloss.Color("196"))

	AddedLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("22")).
			Foreground(lipgloss.Color("46"))

	ModifiedLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("94")).
				Foreground(lipgloss.Color("226"))

	ContextLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	// Line number styles
	LineNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	LineNumberDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("88"))

	LineNumberAddedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("28"))

	// Header and separator styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("87"))

	FilePathStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	HunkHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("87")).
			Bold(true)

	// Word diff styles (for highlighting specific changes within lines)
	WordDeletedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("88")).
				Foreground(lipgloss.Color("224"))

	WordAddedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("28")).
			Foreground(lipgloss.Color("194"))

	// Side-by-side specific styles
	EmptyLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	BorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	// Indicators
	DeleteIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	AddIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)
)

// GetDeletedLineStyle returns the style for deleted lines with optional intensity
func GetDeletedLineStyle(wordLevel bool) lipgloss.Style {
	if wordLevel {
		return WordDeletedStyle
	}
	return DeletedLineStyle
}

// GetAddedLineStyle returns the style for added lines with optional intensity
func GetAddedLineStyle(wordLevel bool) lipgloss.Style {
	if wordLevel {
		return WordAddedStyle
	}
	return AddedLineStyle
}

// ApplyDiffStyle applies the appropriate style based on change type
func ApplyDiffStyle(content string, changeType ChangeType, wordLevel bool) string {
	switch changeType {
	case Delete:
		if wordLevel {
			return WordDeletedStyle.Render(content)
		}
		return DeletedLineStyle.Render(content)
	case Insert:
		if wordLevel {
			return WordAddedStyle.Render(content)
		}
		return AddedLineStyle.Render(content)
	default:
		return ContextLineStyle.Render(content)
	}
}

// FormatLineNumber formats a line number with appropriate styling
func FormatLineNumber(lineNum int, changeType ChangeType) string {
	var style lipgloss.Style
	switch changeType {
	case Delete:
		style = LineNumberDeletedStyle
	case Insert:
		style = LineNumberAddedStyle
	default:
		style = LineNumberStyle
	}
	return style.Render(fmt.Sprintf("%4d", lineNum))
}
