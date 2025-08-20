package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// MessageStatus represents the current state of a message
type MessageStatus string

const (
	MessagePending    MessageStatus = "pending"    // User message waiting to be processed
	MessageProcessing MessageStatus = "processing" // Agent is working on this message
	MessageCompleted  MessageStatus = "completed"  // Agent finished processing
	MessageError      MessageStatus = "error"      // Processing failed
)

// MessageType represents different types of messages
type MessageType string

const (
	MessageTypeText           MessageType = "text"            // Regular user/assistant text message
	MessageTypeToolInvocation MessageType = "tool_invocation" // Tool being invoked
	MessageTypeToolResult     MessageType = "tool_result"     // Tool execution result
)

// Progress represents progress information for a message
type Progress struct {
	Description string  // What the agent is currently doing
	Step        int     // Current step number
	TotalSteps  int     // Total number of steps (if known)
	Percentage  float64 // Completion percentage (0-100)
}

// ToolInfo represents information about a tool invocation
type ToolInfo struct {
	Name       string // Tool name (e.g., "read_file", "edit_file")
	Input      string // Tool input parameters (JSON string)
	Output     string // Tool output/result
	Error      string // Error message if tool failed
	Duration   string // How long the tool took to execute
	ShowOutput bool   // Whether to show the output for this tool
}

// Message represents a chat message
type Message struct {
	ID        string        // Unique identifier for message updates
	Role      string        // "user" or "assistant"
	Content   string        // Message content (can be updated)
	Type      MessageType   // Type of message (text, tool_invocation, tool_result)
	Status    MessageStatus // Current processing status
	IsError   bool          // Legacy error flag
	Timestamp time.Time     // When message was created
	UpdatedAt time.Time     // Last update time
	Progress  *Progress     // Optional progress information
	ToolInfo  *ToolInfo     // Optional tool information for tool-related messages
}

// ShouldShowToolOutput determines if a tool's output should be displayed
func ShouldShowToolOutput(toolName string) bool {
	// Tools that should show output
	outputTools := map[string]bool{
		"edit_file":  true,
		"write_file": true,
		"todoread":   true,
		"todowrite":  true,
		"run_task":   true,
		"ls":         true,
	}

	// Tools that should only show invocation (no output)
	// read_file, and others not listed above

	return outputTools[toolName]
}

// ChatComponent handles the rendering of chat messages
type ChatComponent struct {
	messages       []Message
	height         int
	width          int
	scrollOffset   int  // Lines from bottom (0 = viewing latest)
	focused        bool // Whether the chat window is focused
	cursorLine     int  // Cursor position within visible viewport (0-based)
	visualMode     bool // Whether visual selection is active
	visualLine     bool // true for line mode (V), false for char mode (v)
	selectionStart int  // Starting line of selection (absolute position)
	selectionEnd   int  // Ending line of selection (absolute position)
}

// NewChatComponent creates a new chat component
func NewChatComponent(messages []Message, height int, width int) *ChatComponent {
	return &ChatComponent{
		messages:     messages,
		height:       height,
		width:        width,
		scrollOffset: 0,
		focused:      false,
		cursorLine:   0,
	}
}

// NewChatComponentWithScroll creates a new chat component with scroll position
func NewChatComponentWithScroll(messages []Message, height int, width int, scrollOffset int, focused bool) *ChatComponent {
	return &ChatComponent{
		messages:     messages,
		height:       height,
		width:        width,
		scrollOffset: scrollOffset,
		focused:      focused,
		cursorLine:   0,
	}
}

// NewChatComponentWithCursor creates a new chat component with scroll and cursor position
func NewChatComponentWithCursor(messages []Message, height int, width int, scrollOffset int, focused bool, cursorLine int) *ChatComponent {
	return &ChatComponent{
		messages:     messages,
		height:       height,
		width:        width,
		scrollOffset: scrollOffset,
		focused:      focused,
		cursorLine:   cursorLine,
	}
}

// NewChatComponentWithSelection creates a new chat component with visual selection support
func NewChatComponentWithSelection(messages []Message, height int, width int, scrollOffset int, focused bool, cursorLine int, visualMode bool, visualLine bool, selectionStart int, selectionEnd int) *ChatComponent {
	return &ChatComponent{
		messages:       messages,
		height:         height,
		width:          width,
		scrollOffset:   scrollOffset,
		focused:        focused,
		cursorLine:     cursorLine,
		visualMode:     visualMode,
		visualLine:     visualLine,
		selectionStart: selectionStart,
		selectionEnd:   selectionEnd,
	}
}

// Render renders the chat messages with proper styling and scrolling
func (c *ChatComponent) Render() string {
	result, _ := c.RenderWithSpinners(nil)
	return result
}

// GetSelectedText returns the text currently selected in visual mode
func (c *ChatComponent) GetSelectedText() string {
	if !c.visualMode {
		return ""
	}
	
	// Build all chat message lines (same as in RenderWithSpinners)
	var allLines []string
	for i, msg := range c.messages {
		content := c.renderMessagePlain(msg)
		lines := strings.Split(content, "\n")
		allLines = append(allLines, lines...)
		
		// Add empty line between messages (except after the last message)
		if i < len(c.messages)-1 {
			allLines = append(allLines, "")
		}
	}
	
	// Get selected lines
	minSel := min(c.selectionStart, c.selectionEnd)
	maxSel := max(c.selectionStart, c.selectionEnd)
	
	// Ensure bounds are valid
	minSel = max(0, minSel)
	maxSel = min(len(allLines)-1, maxSel)
	
	if minSel > maxSel || minSel >= len(allLines) {
		return ""
	}
	
	// Extract selected lines
	selectedLines := allLines[minSel : maxSel+1]
	return strings.Join(selectedLines, "\n")
}

// renderMessagePlain renders a message as plain text without styling
func (c *ChatComponent) renderMessagePlain(msg Message) string {
	if msg.Type == MessageTypeToolInvocation || msg.Type == MessageTypeToolResult {
		if msg.ToolInfo != nil {
			if msg.Type == MessageTypeToolInvocation {
				return fmt.Sprintf("Tool: %s", msg.ToolInfo.Name)
			} else {
				return fmt.Sprintf("Tool %s completed", msg.ToolInfo.Name)
			}
		}
		return msg.Content
	}
	
	prefix := ""
	if msg.Role == "user" {
		prefix = "> "
	} else {
		prefix = "⏺ "
	}
	
	return prefix + msg.Content
}

// lineData stores the raw components of a line for deferred rendering
type lineData struct {
	prefix      string
	content     string
	bulletStyle lipgloss.Style
	textStyle   lipgloss.Style
	isEmpty     bool
}

// RenderWithSpinners renders chat messages with spinner support
func (c *ChatComponent) RenderWithSpinners(spinners map[string]*SpinnerComponent) (string, int) {
	// Styles for bullet points only
	userBulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))       // Blue
	assistantBulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // Yellow
	errorBulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))      // Red
	processingBulletStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // Cyan

	// Text style matches input text (default terminal color)
	textStyle := lipgloss.NewStyle() // No color specified, uses default

	// Build all chat message line data (not rendered yet)
	var allLineData []lineData
	for i, msg := range c.messages {
		msgLineData := c.buildMessageLineData(msg, spinners, userBulletStyle, assistantBulletStyle, errorBulletStyle, processingBulletStyle, textStyle)
		allLineData = append(allLineData, msgLineData...)

		// Add empty line between messages (except after the last message)
		if i < len(c.messages)-1 {
			allLineData = append(allLineData, lineData{isEmpty: true})
		}
	}

	totalLines := len(allLineData)
	chatHeight := max(c.height, 1)

	// Calculate visible range based on scroll offset
	// scrollOffset of 0 means we're at the bottom (latest messages)
	// Higher scrollOffset means we're scrolled up to see older messages
	var visibleLineData []lineData
	var startIdx, endIdx int
	
	if totalLines <= chatHeight {
		// All lines fit, no scrolling needed
		visibleLineData = allLineData
		startIdx = 0
		endIdx = totalLines
	} else {
		// Need to apply scrolling
		endIdx = totalLines - c.scrollOffset
		startIdx = max(0, endIdx - chatHeight)
		
		// Ensure we don't go out of bounds
		if endIdx > totalLines {
			endIdx = totalLines
			startIdx = max(0, endIdx - chatHeight)
		}
		if startIdx < 0 {
			startIdx = 0
			endIdx = min(chatHeight, totalLines)
		}
		
		visibleLineData = allLineData[startIdx:endIdx]
	}

	// Add scroll indicators if needed
	var output strings.Builder
	
	// Top indicator if there are lines above the visible window
	if startIdx > 0 {
		moreAbove := startIdx // Number of lines above the visible window
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(c.width).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("↑ %d more lines above ↑", moreAbove))
		output.WriteString(scrollIndicator)
		output.WriteString("\n")
		// Remove one line from visible to make room for indicator
		if len(visibleLineData) > 1 {
			visibleLineData = visibleLineData[1:]
			startIdx++ // Adjust startIdx since we removed first visible line
		}
	}
	
	// Render visible lines with cursor and selection indication if focused
	if c.focused && len(visibleLineData) > 0 {
		// Render lines with cursor and/or selection
		for i, ld := range visibleLineData {
			// Calculate actual line number in full chat
			actualLineNum := i + startIdx
			
			// Check if this line is selected
			isSelected := false
			if c.visualMode {
				minSel := min(c.selectionStart, c.selectionEnd)
				maxSel := max(c.selectionStart, c.selectionEnd)
				isSelected = actualLineNum >= minSel && actualLineNum <= maxSel
			}
			
			// Render the line with appropriate background
			var renderedLine string
			if ld.isEmpty {
				// Empty line
				if i == c.cursorLine {
					cursorStyle := lipgloss.NewStyle().
						Background(lipgloss.Color("237")).
						Width(c.width)
					renderedLine = cursorStyle.Render(" ")
				} else if isSelected && c.visualMode {
					selectionStyle := lipgloss.NewStyle().
						Background(lipgloss.Color("238")).
						Width(c.width)
					renderedLine = selectionStyle.Render(" ")
				} else {
					renderedLine = ""
				}
			} else {
				// Line with content - apply background first, then bullet style
				if isSelected && c.visualMode {
					// Visual selection highlighting
					if i == c.cursorLine {
						// Cursor within selection - brighter background
						backgroundStyle := lipgloss.NewStyle().
							Background(lipgloss.Color("240")).
							Width(c.width)
						// Apply background to entire line, then add styled prefix
						lineContent := ld.prefix + ld.content
						renderedLine = backgroundStyle.Render(lineContent)
						// Apply bullet color on top
						if len(ld.prefix) > 0 {
							prefixWithBg := lipgloss.NewStyle().
								Background(lipgloss.Color("240")).
								Foreground(ld.bulletStyle.GetForeground()).
								Render(ld.prefix)
							contentWithBg := lipgloss.NewStyle().
								Background(lipgloss.Color("240")).
								Render(ld.content)
							renderedLine = prefixWithBg + contentWithBg
						}
					} else {
						// Selection without cursor
						backgroundStyle := lipgloss.NewStyle().
							Background(lipgloss.Color("238")).
							Width(c.width)
						lineContent := ld.prefix + ld.content
						renderedLine = backgroundStyle.Render(lineContent)
						// Apply bullet color on top
						if len(ld.prefix) > 0 {
							prefixWithBg := lipgloss.NewStyle().
								Background(lipgloss.Color("238")).
								Foreground(ld.bulletStyle.GetForeground()).
								Render(ld.prefix)
							contentWithBg := lipgloss.NewStyle().
								Background(lipgloss.Color("238")).
								Render(ld.content)
							renderedLine = prefixWithBg + contentWithBg
						}
					}
				} else if i == c.cursorLine {
					// Just cursor, no selection
					backgroundStyle := lipgloss.NewStyle().
						Background(lipgloss.Color("237")).
						Width(c.width)
					lineContent := ld.prefix + ld.content
					renderedLine = backgroundStyle.Render(lineContent)
					// Apply bullet color on top
					if len(ld.prefix) > 0 {
						prefixWithBg := lipgloss.NewStyle().
							Background(lipgloss.Color("237")).
							Foreground(ld.bulletStyle.GetForeground()).
							Render(ld.prefix)
						contentWithBg := lipgloss.NewStyle().
							Background(lipgloss.Color("237")).
							Render(ld.content)
						renderedLine = prefixWithBg + contentWithBg
					}
				} else {
					// Normal line - apply bullet style then text style
					renderedLine = ld.bulletStyle.Render(ld.prefix) + ld.textStyle.Render(ld.content)
				}
			}
			
			output.WriteString(renderedLine)
			if i < len(visibleLineData)-1 {
				output.WriteString("\n")
			}
		}
	} else {
		// No cursor when not focused - render normally
		for i, ld := range visibleLineData {
			if ld.isEmpty {
				output.WriteString("")
			} else {
				output.WriteString(ld.bulletStyle.Render(ld.prefix) + ld.textStyle.Render(ld.content))
			}
			if i < len(visibleLineData)-1 {
				output.WriteString("\n")
			}
		}
	}
	
	// Bottom indicator if there are more messages below
	moreBelow := totalLines - endIdx
	if moreBelow > 0 {
		// Remove last line to make room for indicator
		lines := strings.Split(output.String(), "\n")
		if len(lines) > 1 {
			output.Reset()
			output.WriteString(strings.Join(lines[:len(lines)-1], "\n"))
		}
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(c.width).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("↓ %d more lines below ↓", moreBelow))
		output.WriteString("\n")
		output.WriteString(scrollIndicator)
	}

	// Pad to fill the viewport height
	currentLines := len(strings.Split(output.String(), "\n"))
	if currentLines < chatHeight {
		paddingLines := chatHeight - currentLines
		output.WriteString(strings.Repeat("\n", paddingLines))
	}

	// Note: Focus indication is handled by border color in the view layer
	// We don't add borders here to avoid double-borders
	return output.String(), totalLines
}

// wrapText wraps text to fit within the specified width, accounting for prefix length
func wrapText(text string, width int, prefixLen int) string {
	if width <= prefixLen {
		return text // Can't wrap meaningfully
	}

	availableWidth := width - prefixLen
	if availableWidth <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= availableWidth {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Wrap this line
		var currentLine strings.Builder
		var currentLen int

		words := strings.Fields(line)
		for i, word := range words {
			wordLen := len(word)
			spaceLen := 0
			if i > 0 {
				spaceLen = 1 // for the space
			}

			// Check if adding this word would exceed the width
			if currentLen+spaceLen+wordLen > availableWidth && currentLen > 0 {
				// Start a new line
				wrappedLines = append(wrappedLines, currentLine.String())
				currentLine.Reset()
				currentLen = 0
				spaceLen = 0
			}

			// Add space if not the first word on the line
			if currentLen > 0 {
				currentLine.WriteString(" ")
				currentLen += 1
			}

			// Add the word
			currentLine.WriteString(word)
			currentLen += wordLen
		}

		// Add any remaining content
		if currentLine.Len() > 0 {
			wrappedLines = append(wrappedLines, currentLine.String())
		}
	}

	return strings.Join(wrappedLines, "\n")
}

// buildMessageLineData builds line data for a message without rendering
func (c *ChatComponent) buildMessageLineData(msg Message, spinners map[string]*SpinnerComponent, userBulletStyle, assistantBulletStyle, errorBulletStyle, processingBulletStyle, textStyle lipgloss.Style) []lineData {
	var lines []lineData
	
	// Handle tool-specific messages
	if msg.Type == MessageTypeToolInvocation || msg.Type == MessageTypeToolResult {
		// For now, render tool messages the old way and convert
		rendered := c.renderToolMessage(msg, spinners, textStyle)
		for _, line := range strings.Split(rendered, "\n") {
			lines = append(lines, lineData{
				prefix:      "",
				content:     line,
				bulletStyle: lipgloss.NewStyle(),
				textStyle:   textStyle,
			})
		}
		return lines
	}

	// Default to text message if Type is empty (backward compatibility)
	if msg.Type == "" {
		msg.Type = MessageTypeText
	}

	var prefix string
	var bulletStyle lipgloss.Style

	// Set role-based prefix and bullet style
	if msg.Role == "user" {
		prefix = "> "
		bulletStyle = userBulletStyle
	} else {
		prefix = "⏺ "
		bulletStyle = assistantBulletStyle
	}

	// Override bullet style for errors
	if msg.IsError || msg.Status == MessageError {
		bulletStyle = errorBulletStyle
	}

	// Set bullet style based on message status
	switch msg.Status {
	case MessagePending:
		bulletStyle = processingBulletStyle
	case MessageProcessing:
		bulletStyle = processingBulletStyle
	case MessageError:
		bulletStyle = errorBulletStyle
	}

	// Build the message content
	var content string
	if msg.Status == MessageProcessing && spinners != nil && spinners[msg.ID] != nil {
		content = msg.Content
		if content == "" && msg.Progress != nil {
			content = msg.Progress.Description
		}
		// For processing messages with spinner, include spinner in prefix
		prefix = prefix + spinners[msg.ID].RenderInline() + " "
	} else {
		content = msg.Content
		if msg.Progress != nil && msg.Status != MessageProcessing {
			content += fmt.Sprintf("\n   %s", msg.Progress.Description)
		}
	}

	// Wrap text accounting for prefix
	wrappedContent := wrapText(content, c.width, len(prefix))
	
	// Handle multi-line content
	contentLines := strings.Split(wrappedContent, "\n")
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}
	
	// First line gets the prefix
	lines = append(lines, lineData{
		prefix:      prefix,
		content:     contentLines[0],
		bulletStyle: bulletStyle,
		textStyle:   textStyle,
	})
	
	// Subsequent lines get indentation
	indent := strings.Repeat(" ", 3)
	for i := 1; i < len(contentLines); i++ {
		lines = append(lines, lineData{
			prefix:      indent,
			content:     contentLines[i],
			bulletStyle: lipgloss.NewStyle(), // No bullet style for indented lines
			textStyle:   textStyle,
		})
	}
	
	return lines
}

// renderMessage renders a single message with appropriate status indicators
func (c *ChatComponent) renderMessage(msg Message, spinners map[string]*SpinnerComponent, userBulletStyle, assistantBulletStyle, errorBulletStyle, processingBulletStyle, textStyle lipgloss.Style) string {
	// Handle tool-specific messages
	if msg.Type == MessageTypeToolInvocation || msg.Type == MessageTypeToolResult {
		return c.renderToolMessage(msg, spinners, textStyle)
	}

	// Default to text message if Type is empty (backward compatibility)
	if msg.Type == "" {
		msg.Type = MessageTypeText
	}

	var prefix string
	var bulletStyle lipgloss.Style

	// Set role-based prefix and bullet style
	if msg.Role == "user" {
		prefix = "> "
		bulletStyle = userBulletStyle
	} else {
		prefix = "⏺ "
		bulletStyle = assistantBulletStyle
	}

	// Override bullet style for errors
	if msg.IsError || msg.Status == MessageError {
		bulletStyle = errorBulletStyle
	}

	// Set bullet style based on message status
	switch msg.Status {
	case MessagePending:
		bulletStyle = processingBulletStyle
	case MessageProcessing:
		bulletStyle = processingBulletStyle
	case MessageError:
		bulletStyle = errorBulletStyle
	}

	// Build the complete message content
	var content string
	if msg.Status == MessageProcessing && spinners != nil && spinners[msg.ID] != nil {
		content = msg.Content
		if content == "" && msg.Progress != nil {
			// For processing messages with no content, show progress inline
			content = msg.Progress.Description
		}

		// Wrap text accounting for bullet + spinner + space
		spinnerPrefix := prefix + spinners[msg.ID].RenderInline() + " "
		wrappedContent := wrapText(content, c.width, len(spinnerPrefix))

		// Handle multi-line content with proper indentation
		lines := strings.Split(wrappedContent, "\n")
		if len(lines) <= 1 {
			return bulletStyle.Render(prefix) + spinners[msg.ID].RenderInline() + " " + textStyle.Render(wrappedContent)
		}

		// First line gets bullet + spinner
		result := bulletStyle.Render(prefix) + spinners[msg.ID].RenderInline() + " " + textStyle.Render(lines[0])
		// Subsequent lines get indentation
		indent := strings.Repeat(" ", 3) // Fixed indentation for visual alignment
		for _, line := range lines[1:] {
			result += "\n" + indent + textStyle.Render(line)
		}
		return result
	} else {
		content = msg.Content
		// Add progress information if available
		if msg.Progress != nil && msg.Status != MessageProcessing {
			content += fmt.Sprintf("\n   %s", msg.Progress.Description)
		}

		// Wrap text accounting for bullet
		wrappedContent := wrapText(content, c.width, len(prefix))

		// Handle multi-line content with proper indentation
		lines := strings.Split(wrappedContent, "\n")
		if len(lines) <= 1 {
			return bulletStyle.Render(prefix) + textStyle.Render(wrappedContent)
		}

		// First line gets bullet
		result := bulletStyle.Render(prefix) + textStyle.Render(lines[0])
		// Subsequent lines get indentation
		indent := strings.Repeat(" ", 3) // Fixed indentation for visual alignment
		for _, line := range lines[1:] {
			result += "\n" + indent + textStyle.Render(line)
		}
		return result
	}
}

// renderToolMessage renders tool invocation and result messages
func (c *ChatComponent) renderToolMessage(msg Message, spinners map[string]*SpinnerComponent, textStyle lipgloss.Style) string {
	// Tool-specific styling
	toolInvocationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // Magenta
	toolSuccessStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))    // Green
	toolErrorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))      // Red
	toolProcessingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // Cyan

	var prefix string
	var bulletStyle lipgloss.Style
	var content string

	if msg.Type == MessageTypeToolInvocation {
		prefix = "🔧 "
		bulletStyle = toolInvocationStyle

		if msg.ToolInfo != nil {
			if msg.Status == MessageProcessing {
				bulletStyle = toolProcessingStyle
				content = fmt.Sprintf("Running tool: %s", msg.ToolInfo.Name)

				// Show input if available (truncated)
				if msg.ToolInfo.Input != "" && msg.ToolInfo.Input != "{}" {
					inputPreview := msg.ToolInfo.Input
					if len(inputPreview) > 100 {
						inputPreview = inputPreview[:100] + "..."
					}
					content += fmt.Sprintf("\n   Input: %s", inputPreview)
				}
			} else {
				content = fmt.Sprintf("Tool: %s", msg.ToolInfo.Name)
			}
		} else {
			content = "Tool invocation"
		}
	} else { // MessageTypeToolResult
		if msg.ToolInfo != nil && msg.ToolInfo.Error != "" {
			prefix = "❌ "
			bulletStyle = toolErrorStyle
			content = fmt.Sprintf("Tool %s failed: %s", msg.ToolInfo.Name, msg.ToolInfo.Error)
		} else {
			prefix = "✅ "
			bulletStyle = toolSuccessStyle
			content = fmt.Sprintf("Tool %s completed", msg.ToolInfo.Name)
			if msg.ToolInfo != nil && msg.ToolInfo.Duration != "" {
				content += fmt.Sprintf(" (%s)", msg.ToolInfo.Duration)
			}

			// Show truncated output if available and tool should show output
			if msg.ToolInfo != nil && msg.ToolInfo.Output != "" && msg.ToolInfo.ShowOutput {
				outputPreview := msg.ToolInfo.Output
				if len(outputPreview) > 200 {
					outputPreview = outputPreview[:200] + "..."
				}
				content += fmt.Sprintf("\n   Result: %s", outputPreview)
			}
		}
	}

	// Handle processing state with spinner
	if msg.Status == MessageProcessing && spinners != nil && spinners[msg.ID] != nil {
		// Wrap text accounting for prefix + spinner + space
		spinnerPrefix := prefix + spinners[msg.ID].RenderInline() + " "
		wrappedContent := wrapText(content, c.width, len(spinnerPrefix))

		// Handle multi-line content with proper indentation
		lines := strings.Split(wrappedContent, "\n")
		if len(lines) <= 1 {
			return bulletStyle.Render(prefix) + spinners[msg.ID].RenderInline() + " " + textStyle.Render(wrappedContent)
		}

		// First line gets prefix + spinner
		result := bulletStyle.Render(prefix) + spinners[msg.ID].RenderInline() + " " + textStyle.Render(lines[0])
		// Subsequent lines get indentation
		indent := strings.Repeat(" ", len(prefix)+2) // prefix + spinner width
		for _, line := range lines[1:] {
			result += "\n" + indent + textStyle.Render(line)
		}
		return result
	} else {
		// Regular rendering without spinner
		wrappedContent := wrapText(content, c.width, len(prefix))

		// Handle multi-line content with proper indentation
		lines := strings.Split(wrappedContent, "\n")
		if len(lines) <= 1 {
			return bulletStyle.Render(prefix) + textStyle.Render(wrappedContent)
		}

		// First line gets prefix
		result := bulletStyle.Render(prefix) + textStyle.Render(lines[0])
		// Subsequent lines get indentation
		indent := strings.Repeat(" ", len(prefix))
		for _, line := range lines[1:] {
			result += "\n" + indent + textStyle.Render(line)
		}
		return result
	}
}
