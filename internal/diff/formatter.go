package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FormatUnified generates a unified diff format output
func FormatUnified(result *DiffResult) string {
	if len(result.Changes) == 0 {
		return ""
	}

	var output strings.Builder
	hunks := groupIntoHunks(result.Changes, result.Context)

	for _, hunk := range hunks {
		output.WriteString(formatHunk(hunk))
		output.WriteString("\n")
	}

	return output.String()
}

// FormatWithLineNumbers formats diff without line numbers, just +/- indicators
func FormatWithLineNumbers(result *DiffResult) string {
	if len(result.Changes) == 0 {
		return ""
	}

	var output strings.Builder
	hunks := groupIntoHunks(result.Changes, result.Context)

	for _, hunk := range hunks {
		// Write hunk header
		output.WriteString(formatHunkHeader(hunk))
		output.WriteString("\n")

		// Write changes without line numbers
		for _, change := range hunk.Changes {
			switch change.Type {
			case Equal:
				for _, line := range change.OldLines {
					output.WriteString("  " + line + "\n")
				}
			case Delete:
				for _, line := range change.OldLines {
					output.WriteString("- " + line + "\n")
				}
			case Insert:
				for _, line := range change.NewLines {
					output.WriteString("+ " + line + "\n")
				}
			}
		}
	}

	return strings.TrimSuffix(output.String(), "\n")
}

// FormatSideBySide generates a side-by-side diff with syntax highlighting
func FormatSideBySide(result *DiffResult, opts SideBySideOptions) string {
	return FormatSideBySideWithSyntax(result, opts)
}

// FormatStyledDiff formats diff with lipgloss styling
func FormatStyledDiff(result *DiffResult) string {
	if len(result.Changes) == 0 {
		return ""
	}

	// Define styles
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // Green
	deletedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // Cyan
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray

	var output strings.Builder
	hunks := groupIntoHunks(result.Changes, result.Context)

	for _, hunk := range hunks {
		// Format hunk header
		header := formatHunkHeader(hunk)
		output.WriteString(headerStyle.Render(header))
		output.WriteString("\n")

		// Calculate max line number for padding
		maxLineNum := 0
		for _, change := range hunk.Changes {
			if change.Type != Insert && change.OldStart+len(change.OldLines)-1 > maxLineNum {
				maxLineNum = change.OldStart + len(change.OldLines) - 1
			}
		}
		width := len(fmt.Sprintf("%d", maxLineNum))

		// Format each change
		for _, change := range hunk.Changes {
			switch change.Type {
			case Equal:
				for i, line := range change.OldLines {
					lineNum := change.OldStart + i
					lineNumStr := fmt.Sprintf("%*d    ", width, lineNum)
					output.WriteString(lineNumStyle.Render(lineNumStr))
					output.WriteString(contextStyle.Render(line))
					output.WriteString("\n")
				}
			case Delete:
				for i, line := range change.OldLines {
					lineNum := change.OldStart + i
					lineNumStr := fmt.Sprintf("%*d -  ", width, lineNum)
					output.WriteString(deletedStyle.Render(lineNumStr + line))
					output.WriteString("\n")
				}
			case Insert:
				for _, line := range change.NewLines {
					lineNumStr := fmt.Sprintf("%*s +  ", width, "")
					output.WriteString(addedStyle.Render(lineNumStr + line))
					output.WriteString("\n")
				}
			}
		}
	}

	return output.String()
}

// Hunk represents a group of changes with context
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Changes  []Change
}

// groupIntoHunks groups changes into hunks with context
func groupIntoHunks(changes []Change, contextSize int) []Hunk {
	if len(changes) == 0 {
		return []Hunk{}
	}

	var hunks []Hunk
	var pendingChanges []Change
	inChangeBlock := false
	var contextBeforeLines []Change // Store individual line changes
	var contextAfterLines []Change  // Store individual line changes

	// Helper to extract last N lines from Equal changes
	extractLastLines := func(changes []Change, n int) []Change {
		var result []Change
		totalLines := 0

		// Count total lines in all changes
		for _, c := range changes {
			if c.Type == Equal {
				totalLines += len(c.OldLines)
			}
		}

		// If we have fewer lines than needed, return all
		if totalLines <= n {
			return changes
		}

		// Extract last n lines
		skip := totalLines - n
		for _, c := range changes {
			if c.Type == Equal {
				if skip >= len(c.OldLines) {
					skip -= len(c.OldLines)
					continue
				}

				// Take lines from this change
				startIdx := skip
				if startIdx < 0 {
					startIdx = 0
				}

				if startIdx < len(c.OldLines) {
					newChange := Change{
						Type:     Equal,
						OldStart: c.OldStart + startIdx,
						OldLines: c.OldLines[startIdx:],
						NewStart: c.NewStart + startIdx,
						NewLines: c.NewLines[startIdx:],
					}
					result = append(result, newChange)
				}
				skip = 0
			}
		}
		return result
	}

	// Helper to extract first N lines from Equal changes
	extractFirstLines := func(changes []Change, n int) []Change {
		var result []Change
		remaining := n

		for _, c := range changes {
			if remaining <= 0 {
				break
			}

			if c.Type == Equal {
				if len(c.OldLines) <= remaining {
					// Take entire change
					result = append(result, c)
					remaining -= len(c.OldLines)
				} else {
					// Take only first 'remaining' lines
					newChange := Change{
						Type:     Equal,
						OldStart: c.OldStart,
						OldLines: c.OldLines[:remaining],
						NewStart: c.NewStart,
						NewLines: c.NewLines[:remaining],
					}
					result = append(result, newChange)
					remaining = 0
				}
			}
		}
		return result
	}

	// Helper to count total lines in Equal changes
	countEqualLines := func(changes []Change) int {
		count := 0
		for _, c := range changes {
			if c.Type == Equal {
				count += len(c.OldLines)
			}
		}
		return count
	}

	for i, change := range changes {
		if change.Type != Equal {
			// This is an actual change (Insert or Delete)
			if !inChangeBlock {
				// Starting a new change block
				inChangeBlock = true
				// Add up to contextSize lines of context before
				contextChanges := extractLastLines(contextBeforeLines, contextSize)
				pendingChanges = append(pendingChanges, contextChanges...)
				contextBeforeLines = []Change{} // Clear context before
			}
			pendingChanges = append(pendingChanges, change)
			contextAfterLines = []Change{} // Reset context after when we see a change
		} else {
			// This is an Equal (context) line
			if inChangeBlock {
				// We're in a change block, this might be trailing context
				contextAfterLines = append(contextAfterLines, change)
				totalAfterLines := countEqualLines(contextAfterLines)

				// Check if we've collected enough trailing context or reached the end
				if totalAfterLines >= contextSize || i == len(changes)-1 {
					// Add the trailing context to pending changes
					contextChanges := extractFirstLines(contextAfterLines, contextSize)
					pendingChanges = append(pendingChanges, contextChanges...)

					// Check if we have more context than needed (potential hunk split)
					if totalAfterLines > contextSize*2 {
						// We have enough context to end this hunk
						// Create the hunk from pending changes
						if len(pendingChanges) > 0 {
							hunk := createHunkFromChanges(pendingChanges)
							hunks = append(hunks, hunk)
						}
						// Reset for next hunk
						pendingChanges = []Change{}
						inChangeBlock = false
						// Save remaining context for potential next hunk
						// Skip the first contextSize lines we already used
						var remainingContext []Change
						skipped := contextSize
						for _, c := range contextAfterLines {
							if c.Type == Equal {
								if skipped >= len(c.OldLines) {
									skipped -= len(c.OldLines)
								} else if skipped > 0 {
									// Partial skip
									newChange := Change{
										Type:     Equal,
										OldStart: c.OldStart + skipped,
										OldLines: c.OldLines[skipped:],
										NewStart: c.NewStart + skipped,
										NewLines: c.NewLines[skipped:],
									}
									remainingContext = append(remainingContext, newChange)
									skipped = 0
								} else {
									remainingContext = append(remainingContext, c)
								}
							}
						}
						contextBeforeLines = remainingContext
						contextAfterLines = []Change{}
					}
				}
			} else {
				// Not in a change block, accumulate as potential leading context
				contextBeforeLines = append(contextBeforeLines, change)
			}
		}
	}

	// Process any remaining pending changes
	if len(pendingChanges) > 0 {
		hunk := createHunkFromChanges(pendingChanges)
		hunks = append(hunks, hunk)
	}

	return hunks
}

// createHunkFromChanges creates a hunk from a list of changes
func createHunkFromChanges(changes []Change) Hunk {
	if len(changes) == 0 {
		return Hunk{}
	}

	hunk := Hunk{
		OldStart: changes[0].OldStart,
		NewStart: changes[0].NewStart,
		Changes:  changes,
	}

	// Calculate counts
	for _, change := range changes {
		switch change.Type {
		case Equal:
			hunk.OldCount += len(change.OldLines)
			hunk.NewCount += len(change.NewLines)
		case Delete:
			hunk.OldCount += len(change.OldLines)
		case Insert:
			hunk.NewCount += len(change.NewLines)
		}
	}

	return hunk
}

// formatHunk formats a single hunk
func formatHunk(hunk Hunk) string {
	var output strings.Builder

	// Write hunk header
	output.WriteString(formatHunkHeader(hunk))
	output.WriteString("\n")

	// Write changes
	for _, change := range hunk.Changes {
		switch change.Type {
		case Equal:
			for _, line := range change.OldLines {
				output.WriteString(" " + line + "\n")
			}
		case Delete:
			for _, line := range change.OldLines {
				output.WriteString("-" + line + "\n")
			}
		case Insert:
			for _, line := range change.NewLines {
				output.WriteString("+" + line + "\n")
			}
		}
	}

	return output.String()
}

// formatHunkWithLineNumbers formats a hunk with line numbers
func formatHunkWithLineNumbers(hunk Hunk) string {
	var output strings.Builder

	// Write hunk header
	output.WriteString(formatHunkHeader(hunk))
	output.WriteString("\n")

	// Calculate max line number for padding
	maxLineNum := 0
	for _, change := range hunk.Changes {
		if change.Type != Insert && change.OldStart+len(change.OldLines)-1 > maxLineNum {
			maxLineNum = change.OldStart + len(change.OldLines) - 1
		}
	}
	width := len(fmt.Sprintf("%d", maxLineNum))

	// Write changes with line numbers
	for _, change := range hunk.Changes {
		switch change.Type {
		case Equal:
			for i, line := range change.OldLines {
				lineNum := change.OldStart + i
				output.WriteString(fmt.Sprintf("%*d    %s\n", width, lineNum, line))
			}
		case Delete:
			for i, line := range change.OldLines {
				lineNum := change.OldStart + i
				output.WriteString(fmt.Sprintf("%*d -  %s\n", width, lineNum, line))
			}
		case Insert:
			for _, line := range change.NewLines {
				output.WriteString(fmt.Sprintf("%*s +  %s\n", width, "", line))
			}
		}
	}

	return output.String()
}

// formatHunkHeader formats the hunk header line
func formatHunkHeader(hunk Hunk) string {
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)
}
