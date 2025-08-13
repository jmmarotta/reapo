package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FormatSideBySideWithSyntax generates a side-by-side diff view with syntax highlighting
func FormatSideBySideWithSyntax(result *DiffResult, opts SideBySideOptions) string {
	if len(result.Changes) == 0 {
		return ""
	}

	// Set defaults
	if opts.ColumnWidth == 0 {
		opts.ColumnWidth = 80
	}

	// Syntax highlighting has been removed from this project

	var output strings.Builder

	// Add file header or top border
	if opts.ShowFileHeader && opts.FilePath != "" {
		header := formatFileHeader(opts.FilePath, opts.ColumnWidth*2+10)
		output.WriteString(header)
		output.WriteString("\n")
	} else {
		// Add simple top border without file path
		output.WriteString(formatTopBorder(opts.ColumnWidth))
		output.WriteString("\n")
	}

	// Group changes into hunks
	hunks := groupIntoHunks(result.Changes, result.Context)

	for hunkIdx, hunk := range hunks {
		if hunkIdx > 0 {
			// Add hunk separator
			output.WriteString(formatHunkSeparator(opts.ColumnWidth))
			output.WriteString("\n")
		}

		// Process each change in the hunk
		oldLineNum := hunk.OldStart
		newLineNum := hunk.NewStart

		for _, change := range hunk.Changes {
			switch change.Type {
			case Equal:
				// Context lines - show on both sides
				for i, line := range change.OldLines {
					leftWrapped := formatContextLines(oldLineNum, line, opts)
					rightWrapped := formatContextLines(newLineNum, change.NewLines[i], opts)
					
					// Ensure both sides have the same number of lines for alignment
					maxLines := len(leftWrapped)
					if len(rightWrapped) > maxLines {
						maxLines = len(rightWrapped)
					}
					
					for j := 0; j < maxLines; j++ {
						var leftContent, rightContent string
						if j < len(leftWrapped) {
							leftContent = leftWrapped[j]
						} else {
							leftContent = formatEmptyLine(opts)
						}
						if j < len(rightWrapped) {
							rightContent = rightWrapped[j]
						} else {
							rightContent = formatEmptyLine(opts)
						}
						output.WriteString(formatSideBySideLine(leftContent, rightContent, opts.ColumnWidth))
						output.WriteString("\n")
					}
					oldLineNum++
					newLineNum++
				}

			case Delete:
				// Deletion - show on left side only
				for _, line := range change.OldLines {
					leftWrapped := formatDeletedLines(oldLineNum, line, opts)
					
					for _, leftContent := range leftWrapped {
						rightContent := formatEmptyLine(opts)
						output.WriteString(formatSideBySideLine(leftContent, rightContent, opts.ColumnWidth))
						output.WriteString("\n")
					}
					oldLineNum++
				}

			case Insert:
				// Addition - show on right side only
				for _, line := range change.NewLines {
					rightWrapped := formatAddedLines(newLineNum, line, opts)
					
					for _, rightContent := range rightWrapped {
						leftContent := formatEmptyLine(opts)
						output.WriteString(formatSideBySideLine(leftContent, rightContent, opts.ColumnWidth))
						output.WriteString("\n")
					}
					newLineNum++
				}
			}
		}
	}

	// Add bottom border
	output.WriteString(formatBottomBorder(opts.ColumnWidth))

	return output.String()
}

// formatTopBorder creates a simple top border without file path
func formatTopBorder(columnWidth int) string {
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray
	left := strings.Repeat("─", columnWidth+6)
	right := strings.Repeat("─", columnWidth+6)
	return borderStyle.Render("╭" + left + "┬" + right + "╮")
}

// formatFileHeader creates the top border with file path
func formatFileHeader(filePath string, width int) string {
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray for borders
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	// Create the header with file path
	header := fmt.Sprintf("─ %s ", filePath)
	remainingWidth := width - len(header) - 2
	if remainingWidth < 0 {
		remainingWidth = 0
	}
	header += strings.Repeat("─", remainingWidth)

	return borderStyle.Render("╭") + headerStyle.Render(header) + borderStyle.Render("╮")
}

// formatColumnSeparator creates the separator between columns
func formatColumnSeparator(columnWidth int) string {
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray
	left := strings.Repeat("─", columnWidth+6)
	right := strings.Repeat("─", columnWidth+6)
	return separatorStyle.Render("├" + left + "┬" + right + "┤")
}

// formatHunkSeparator creates a separator between hunks
func formatHunkSeparator(columnWidth int) string {
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray
	left := strings.Repeat("·", columnWidth+6)
	right := strings.Repeat("·", columnWidth+6)
	return separatorStyle.Render("│" + left + "│" + right + "│")
}

// formatBottomBorder creates the bottom border
func formatBottomBorder(columnWidth int) string {
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray
	left := strings.Repeat("─", columnWidth+6)
	right := strings.Repeat("─", columnWidth+6)
	return separatorStyle.Render("╰" + left + "┴" + right + "╯")
}

// formatSideBySideLine formats a single line in side-by-side view
func formatSideBySideLine(left, right string, columnWidth int) string {
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray

	// Ensure proper width for each column
	leftPadded := padToWidth(left, columnWidth+6)
	rightPadded := padToWidth(right, columnWidth+6)

	// Render each vertical bar separately to maintain consistent border color
	return separatorStyle.Render("│") + leftPadded + separatorStyle.Render("│") + rightPadded + separatorStyle.Render("│")
}

// formatContextLine formats a context line (returns multiple lines if wrapped)
func formatContextLines(lineNum int, content string, opts SideBySideOptions) []string {
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray for line numbers
	
	var contentWidth int
	if opts.ShowLineNumbers {
		contentWidth = opts.ColumnWidth
	} else {
		contentWidth = opts.ColumnWidth + 5
	}
	
	wrappedLines := wrapString(content, contentWidth)
	var result []string
	
	for i, line := range wrappedLines {
		if opts.ShowLineNumbers {
			if i == 0 {
				// First line shows the line number
				lineNumStr := fmt.Sprintf("%4d  ", lineNum)
				result = append(result, lineNumStyle.Render(lineNumStr) + line)
			} else {
				// Continuation lines have no line number
				result = append(result, "      " + line)
			}
		} else {
			result = append(result, " " + line)
		}
	}
	
	return result
}

// formatDeletedLine formats a deleted line (returns multiple lines if wrapped)
func formatDeletedLines(lineNum int, content string, opts SideBySideOptions) []string {
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray for line numbers
	deletedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Background(lipgloss.Color("52"))
	
	var contentWidth int
	if opts.ShowLineNumbers {
		contentWidth = opts.ColumnWidth
	} else {
		contentWidth = opts.ColumnWidth + 5
	}
	
	wrappedLines := wrapString(content, contentWidth)
	var result []string
	
	for i, line := range wrappedLines {
		if opts.ShowLineNumbers {
			if i == 0 {
				// First line shows the line number
				lineNumStr := fmt.Sprintf("%4d  ", lineNum)
				result = append(result, lineNumStyle.Render(lineNumStr) + deletedStyle.Render(line))
			} else {
				// Continuation lines have no line number but maintain the style
				result = append(result, "      " + deletedStyle.Render(line))
			}
		} else {
			result = append(result, deletedStyle.Render(" " + line))
		}
	}
	
	return result
}

// formatAddedLine formats an added line (returns multiple lines if wrapped)
func formatAddedLines(lineNum int, content string, opts SideBySideOptions) []string {
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // Light gray for line numbers
	addedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Background(lipgloss.Color("22"))
	
	var contentWidth int
	if opts.ShowLineNumbers {
		contentWidth = opts.ColumnWidth
	} else {
		contentWidth = opts.ColumnWidth + 5
	}
	
	wrappedLines := wrapString(content, contentWidth)
	var result []string
	
	for i, line := range wrappedLines {
		if opts.ShowLineNumbers {
			if i == 0 {
				// First line shows the line number
				lineNumStr := fmt.Sprintf("%4d  ", lineNum)
				result = append(result, lineNumStyle.Render(lineNumStr) + addedStyle.Render(line))
			} else {
				// Continuation lines have no line number but maintain the style
				result = append(result, "      " + addedStyle.Render(line))
			}
		} else {
			result = append(result, addedStyle.Render(" " + line))
		}
	}
	
	return result
}

// formatEmptyLine formats an empty line (for additions/deletions)
func formatEmptyLine(opts SideBySideOptions) string {
	if opts.ShowLineNumbers {
		return strings.Repeat(" ", opts.ColumnWidth+6)
	}
	return strings.Repeat(" ", opts.ColumnWidth+6)
}

// wrapString wraps a string to fit within the given width
func wrapString(s string, width int) []string {
	if len(s) <= width {
		return []string{s}
	}
	
	var lines []string
	for len(s) > 0 {
		if len(s) <= width {
			lines = append(lines, s)
			break
		}
		// Find a good break point (prefer spaces)
		breakPoint := width
		for i := width - 1; i > width*2/3; i-- {
			if s[i] == ' ' {
				breakPoint = i + 1 // Include the space at the end of the line
				break
			}
		}
		lines = append(lines, s[:breakPoint])
		s = s[breakPoint:]
		// Trim leading space from continuation line
		if len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
	}
	return lines
}

// padToWidth pads a string to the specified width
func padToWidth(s string, width int) string {
	// Remove ANSI escape sequences for length calculation
	plainLen := lipgloss.Width(s)
	if plainLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-plainLen)
}
