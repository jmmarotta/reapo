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
	contextBefore := []Change{}
	contextAfter := []Change{}
	
	for i, change := range changes {
		if change.Type != Equal {
			// This is an actual change (Insert or Delete)
			if !inChangeBlock {
				// Starting a new change block
				inChangeBlock = true
				// Add up to contextSize lines of context before
				startIdx := len(contextBefore) - contextSize
				if startIdx < 0 {
					startIdx = 0
				}
				for j := startIdx; j < len(contextBefore); j++ {
					pendingChanges = append(pendingChanges, contextBefore[j])
				}
				contextBefore = []Change{} // Clear context before
			}
			pendingChanges = append(pendingChanges, change)
			contextAfter = []Change{} // Reset context after when we see a change
		} else {
			// This is an Equal (context) line
			if inChangeBlock {
				// We're in a change block, this might be trailing context
				contextAfter = append(contextAfter, change)
				
				// Check if we've collected enough trailing context or reached the end
				if len(contextAfter) >= contextSize || i == len(changes)-1 {
					// Add the trailing context to pending changes
					for j := 0; j < contextSize && j < len(contextAfter); j++ {
						pendingChanges = append(pendingChanges, contextAfter[j])
					}
					
					// Check if we have more context than needed (potential hunk split)
					if len(contextAfter) > contextSize*2 {
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
						contextBefore = contextAfter[contextSize:]
						contextAfter = []Change{}
					}
				}
			} else {
				// Not in a change block, accumulate as potential leading context
				contextBefore = append(contextBefore, change)
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