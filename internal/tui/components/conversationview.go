package components

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
)

// SelectionMode represents the type of selection in conversation view
type SelectionMode int

const (
	SelectionNone SelectionMode = iota
	SelectionChar
	SelectionLine
	SelectionBlock
)

// SearchDirection represents the direction of search
type SearchDirection int

const (
	SearchForward SearchDirection = iota
	SearchBackward
)

// Position represents a cursor position in the conversation view
type Position struct {
	Line int
	Col  int
}

// RenderedLine represents a single rendered line in the conversation view
type RenderedLine struct {
	Content      string
	MessageIndex int     // Which message this line belongs to
	IsUserMsg    bool    // For styling
	IsAssistant  bool
	IsError      bool
	Highlights   []Range // Search match highlights
}

// Range represents a text range for highlighting
type Range struct {
	Start int
	End   int
}

// ConversationView manages the scrollable conversation history view
type ConversationView struct {
	// Core data
	messages []Message // Full message history
	width    int       // Terminal width
	height   int       // Viewport height

	// Cursor and viewport state
	cursorLine  int // Current cursor line position (0-based)
	cursorCol   int // Current cursor column position
	viewportTop int // First visible line index

	// Selection state
	SelectionMode  SelectionMode // None, Char, Line, Block (exported for access)
	selectionStart Position      // Start of selection
	selectionEnd   Position      // End of selection (cursor position)

	// Search state
	searchPattern   string          // Current search pattern
	searchMatches   []Position      // All match positions
	currentMatch    int             // Current match index
	searchDirection SearchDirection // Forward or Backward
	searchMode      bool            // Whether we're in search mode

	// Rendering cache
	renderedLines []RenderedLine // Pre-processed lines with formatting
	lineOffsets   []int          // Message index for each line
	totalLines    int            // Total number of rendered lines

	// Yank buffer
	yankBuffer string // Last yanked text

	// Mode tracking
	needsRedraw bool // Whether the view needs to be re-rendered
}

// NewConversationView creates a new conversation view component
func NewConversationView(messages []Message, width, height int) *ConversationView {
	cv := &ConversationView{
		messages:      messages,
		width:         width,
		height:        height,
		cursorLine:    0,
		cursorCol:     0,
		viewportTop:   0,
		SelectionMode: SelectionNone,
		searchMode:    false,
		needsRedraw:   true,
	}

	cv.preprocessMessages()
	// Start at the bottom of the conversation
	if cv.totalLines > 0 {
		cv.cursorLine = cv.totalLines - 1
		cv.updateViewport()
	}

	return cv
}

// preprocessMessages converts messages into rendered lines
func (cv *ConversationView) preprocessMessages() {
	cv.renderedLines = []RenderedLine{}
	cv.lineOffsets = []int{}

	for msgIdx, msg := range cv.messages {
		// Split message content into lines
		lines := strings.Split(msg.Content, "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}

		for _, line := range lines {
			// Wrap long lines if needed
			wrappedLines := cv.wrapLine(line, cv.width-4) // Account for line numbers/margins

			for _, wrappedLine := range wrappedLines {
				renderedLine := RenderedLine{
					Content:      wrappedLine,
					MessageIndex: msgIdx,
					IsUserMsg:    msg.Role == "user",
					IsAssistant:  msg.Role == "assistant",
					IsError:      msg.IsError || msg.Status == MessageError,
				}

				cv.renderedLines = append(cv.renderedLines, renderedLine)
				cv.lineOffsets = append(cv.lineOffsets, msgIdx)
			}
		}

		// Add empty line between messages
		if msgIdx < len(cv.messages)-1 {
			cv.renderedLines = append(cv.renderedLines, RenderedLine{
				Content:      "",
				MessageIndex: msgIdx,
			})
			cv.lineOffsets = append(cv.lineOffsets, msgIdx)
		}
	}

	cv.totalLines = len(cv.renderedLines)
}

// wrapLine wraps a line to fit within the specified width
func (cv *ConversationView) wrapLine(line string, width int) []string {
	if width <= 0 || len(line) <= width {
		return []string{line}
	}

	var wrapped []string
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	currentLine := ""
	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			wrapped = append(wrapped, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		wrapped = append(wrapped, currentLine)
	}

	return wrapped
}

// updateViewport ensures the cursor is visible in the viewport
func (cv *ConversationView) updateViewport() {
	// Ensure cursor is within bounds
	if cv.cursorLine < 0 {
		cv.cursorLine = 0
	}
	if cv.cursorLine >= cv.totalLines {
		cv.cursorLine = cv.totalLines - 1
	}

	// Adjust viewport to keep cursor visible
	if cv.cursorLine < cv.viewportTop {
		cv.viewportTop = cv.cursorLine
	} else if cv.cursorLine >= cv.viewportTop+cv.height {
		cv.viewportTop = cv.cursorLine - cv.height + 1
	}

	// Ensure viewport is within bounds
	if cv.viewportTop < 0 {
		cv.viewportTop = 0
	}
	maxViewportTop := cv.totalLines - cv.height
	if maxViewportTop < 0 {
		maxViewportTop = 0
	}
	if cv.viewportTop > maxViewportTop {
		cv.viewportTop = maxViewportTop
	}
}

// Navigation methods

// MoveUp moves the cursor up by count lines
func (cv *ConversationView) MoveUp(count int) {
	cv.cursorLine -= count
	if cv.cursorLine < 0 {
		cv.cursorLine = 0
	}
	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveDown moves the cursor down by count lines
func (cv *ConversationView) MoveDown(count int) {
	cv.cursorLine += count
	if cv.cursorLine >= cv.totalLines {
		cv.cursorLine = cv.totalLines - 1
	}
	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveLeft moves the cursor left by count characters
func (cv *ConversationView) MoveLeft(count int) {
	cv.cursorCol -= count
	if cv.cursorCol < 0 {
		cv.cursorCol = 0
	}
	cv.needsRedraw = true
}

// MoveRight moves the cursor right by count characters
func (cv *ConversationView) MoveRight(count int) {
	if cv.cursorLine < len(cv.renderedLines) {
		lineLen := len(cv.renderedLines[cv.cursorLine].Content)
		cv.cursorCol += count
		if cv.cursorCol > lineLen {
			cv.cursorCol = lineLen
		}
	}
	cv.needsRedraw = true
}

// MoveToLineStart moves cursor to beginning of line
func (cv *ConversationView) MoveToLineStart() {
	cv.cursorCol = 0
	cv.needsRedraw = true
}

// MoveToLineEnd moves cursor to end of line
func (cv *ConversationView) MoveToLineEnd() {
	if cv.cursorLine < len(cv.renderedLines) {
		cv.cursorCol = len(cv.renderedLines[cv.cursorLine].Content)
	}
	cv.needsRedraw = true
}

// MoveToFirstNonBlank moves cursor to first non-blank character
func (cv *ConversationView) MoveToFirstNonBlank() {
	if cv.cursorLine < len(cv.renderedLines) {
		line := cv.renderedLines[cv.cursorLine].Content
		for i, ch := range line {
			if ch != ' ' && ch != '\t' {
				cv.cursorCol = i
				cv.needsRedraw = true
				return
			}
		}
	}
	cv.cursorCol = 0
	cv.needsRedraw = true
}

// MoveToFirstLine jumps to the first line
func (cv *ConversationView) MoveToFirstLine() {
	cv.cursorLine = 0
	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveToLastLine jumps to the last line
func (cv *ConversationView) MoveToLastLine() {
	cv.cursorLine = cv.totalLines - 1
	cv.updateViewport()
	cv.needsRedraw = true
}

// PageUp scrolls up one page
func (cv *ConversationView) PageUp() {
	cv.cursorLine -= cv.height
	if cv.cursorLine < 0 {
		cv.cursorLine = 0
	}
	cv.updateViewport()
	cv.needsRedraw = true
}

// PageDown scrolls down one page
func (cv *ConversationView) PageDown() {
	cv.cursorLine += cv.height
	if cv.cursorLine >= cv.totalLines {
		cv.cursorLine = cv.totalLines - 1
	}
	cv.updateViewport()
	cv.needsRedraw = true
}

// HalfPageUp scrolls up half a page
func (cv *ConversationView) HalfPageUp() {
	cv.cursorLine -= cv.height / 2
	if cv.cursorLine < 0 {
		cv.cursorLine = 0
	}
	cv.updateViewport()
	cv.needsRedraw = true
}

// HalfPageDown scrolls down half a page
func (cv *ConversationView) HalfPageDown() {
	cv.cursorLine += cv.height / 2
	if cv.cursorLine >= cv.totalLines {
		cv.cursorLine = cv.totalLines - 1
	}
	cv.updateViewport()
	cv.needsRedraw = true
}

// ScrollUp scrolls the viewport up without moving cursor
func (cv *ConversationView) ScrollUp() {
	cv.viewportTop--
	if cv.viewportTop < 0 {
		cv.viewportTop = 0
	}
	// Adjust cursor if it goes out of view
	if cv.cursorLine >= cv.viewportTop+cv.height {
		cv.cursorLine = cv.viewportTop + cv.height - 1
	}
	cv.needsRedraw = true
}

// ScrollDown scrolls the viewport down without moving cursor
func (cv *ConversationView) ScrollDown() {
	maxViewportTop := cv.totalLines - cv.height
	if maxViewportTop < 0 {
		maxViewportTop = 0
	}
	cv.viewportTop++
	if cv.viewportTop > maxViewportTop {
		cv.viewportTop = maxViewportTop
	}
	// Adjust cursor if it goes out of view
	if cv.cursorLine < cv.viewportTop {
		cv.cursorLine = cv.viewportTop
	}
	cv.needsRedraw = true
}

// Word navigation helpers

// isWordChar checks if a character is part of a word
func isWordChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isWhitespace checks if a character is whitespace
func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// MoveWordForward moves to the beginning of the next word
func (cv *ConversationView) MoveWordForward() {
	if cv.cursorLine >= len(cv.renderedLines) {
		return
	}

	line := cv.renderedLines[cv.cursorLine].Content
	startedOnWord := false
	
	// If we're at the end of the current line, move to next line
	if cv.cursorCol >= len(line) {
		if cv.cursorLine < cv.totalLines-1 {
			cv.cursorLine++
			cv.cursorCol = 0
			// Skip to first non-whitespace
			if cv.cursorLine < len(cv.renderedLines) {
				newLine := cv.renderedLines[cv.cursorLine].Content
				for cv.cursorCol < len(newLine) && isWhitespace(rune(newLine[cv.cursorCol])) {
					cv.cursorCol++
				}
			}
		}
		cv.updateViewport()
		cv.needsRedraw = true
		return
	}

	// Check if we're starting on a word character
	if cv.cursorCol < len(line) {
		startedOnWord = isWordChar(rune(line[cv.cursorCol]))
	}

	// Skip current word if we're in one
	if startedOnWord {
		for cv.cursorCol < len(line) && isWordChar(rune(line[cv.cursorCol])) {
			cv.cursorCol++
		}
	}

	// Skip non-word characters
	for cv.cursorCol < len(line) && !isWordChar(rune(line[cv.cursorCol])) && !isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol++
	}

	// Skip whitespace
	for cv.cursorCol < len(line) && isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol++
	}

	// If we've reached the end of the line, move to next line
	if cv.cursorCol >= len(line) && cv.cursorLine < cv.totalLines-1 {
		cv.cursorLine++
		cv.cursorCol = 0
		// Skip to first non-whitespace on new line
		if cv.cursorLine < len(cv.renderedLines) {
			newLine := cv.renderedLines[cv.cursorLine].Content
			for cv.cursorCol < len(newLine) && isWhitespace(rune(newLine[cv.cursorCol])) {
				cv.cursorCol++
			}
		}
	}

	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveWordBackward moves to the beginning of the previous word
func (cv *ConversationView) MoveWordBackward() {
	if cv.cursorLine >= len(cv.renderedLines) {
		return
	}

	// If at the beginning of a line, move to end of previous line
	if cv.cursorCol == 0 {
		if cv.cursorLine > 0 {
			cv.cursorLine--
			if cv.cursorLine < len(cv.renderedLines) {
				cv.cursorCol = len(cv.renderedLines[cv.cursorLine].Content)
				if cv.cursorCol > 0 {
					cv.cursorCol-- // Move to last character
				}
			}
		}
		cv.updateViewport()
		cv.needsRedraw = true
		return
	}

	line := cv.renderedLines[cv.cursorLine].Content
	cv.cursorCol--

	// Skip whitespace backwards
	for cv.cursorCol > 0 && isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol--
	}

	// If we're on a word character, skip to beginning of word
	if cv.cursorCol >= 0 && isWordChar(rune(line[cv.cursorCol])) {
		for cv.cursorCol > 0 && isWordChar(rune(line[cv.cursorCol-1])) {
			cv.cursorCol--
		}
	} else {
		// Skip non-word, non-whitespace characters
		for cv.cursorCol > 0 && !isWordChar(rune(line[cv.cursorCol])) && !isWhitespace(rune(line[cv.cursorCol])) {
			cv.cursorCol--
		}
		// Then skip the word
		for cv.cursorCol > 0 && isWordChar(rune(line[cv.cursorCol-1])) {
			cv.cursorCol--
		}
	}

	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveWordEnd moves to the end of the current/next word
func (cv *ConversationView) MoveWordEnd() {
	if cv.cursorLine >= len(cv.renderedLines) {
		return
	}

	line := cv.renderedLines[cv.cursorLine].Content
	
	// If at end of line, move to next line
	if cv.cursorCol >= len(line) {
		if cv.cursorLine < cv.totalLines-1 {
			cv.cursorLine++
			cv.cursorCol = 0
			if cv.cursorLine < len(cv.renderedLines) {
				line = cv.renderedLines[cv.cursorLine].Content
			}
		} else {
			return
		}
	}

	// Move forward at least one character
	if cv.cursorCol < len(line)-1 {
		cv.cursorCol++
	}

	// Skip whitespace
	for cv.cursorCol < len(line) && isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol++
	}

	// Move to end of word
	if cv.cursorCol < len(line) && isWordChar(rune(line[cv.cursorCol])) {
		for cv.cursorCol < len(line)-1 && isWordChar(rune(line[cv.cursorCol+1])) {
			cv.cursorCol++
		}
	} else {
		// Move to end of non-word characters
		for cv.cursorCol < len(line)-1 && !isWordChar(rune(line[cv.cursorCol+1])) && !isWhitespace(rune(line[cv.cursorCol+1])) {
			cv.cursorCol++
		}
	}

	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveBigWordForward moves to the beginning of the next WORD (space-delimited)
func (cv *ConversationView) MoveBigWordForward() {
	if cv.cursorLine >= len(cv.renderedLines) {
		return
	}

	line := cv.renderedLines[cv.cursorLine].Content
	
	// Skip current non-whitespace
	for cv.cursorCol < len(line) && !isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol++
	}

	// Skip whitespace
	for cv.cursorCol < len(line) && isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol++
	}

	// If at end of line, move to next line
	if cv.cursorCol >= len(line) && cv.cursorLine < cv.totalLines-1 {
		cv.cursorLine++
		cv.cursorCol = 0
		// Skip leading whitespace on new line
		if cv.cursorLine < len(cv.renderedLines) {
			newLine := cv.renderedLines[cv.cursorLine].Content
			for cv.cursorCol < len(newLine) && isWhitespace(rune(newLine[cv.cursorCol])) {
				cv.cursorCol++
			}
		}
	}

	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveBigWordBackward moves to the beginning of the previous WORD (space-delimited)
func (cv *ConversationView) MoveBigWordBackward() {
	if cv.cursorLine >= len(cv.renderedLines) {
		return
	}

	// If at beginning of line, move to end of previous line
	if cv.cursorCol == 0 {
		if cv.cursorLine > 0 {
			cv.cursorLine--
			if cv.cursorLine < len(cv.renderedLines) {
				cv.cursorCol = len(cv.renderedLines[cv.cursorLine].Content)
			}
		}
		cv.updateViewport()
		cv.needsRedraw = true
		return
	}

	line := cv.renderedLines[cv.cursorLine].Content
	cv.cursorCol--

	// Skip whitespace backwards
	for cv.cursorCol > 0 && isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol--
	}

	// Skip non-whitespace backwards
	for cv.cursorCol > 0 && !isWhitespace(rune(line[cv.cursorCol-1])) {
		cv.cursorCol--
	}

	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveBigWordEnd moves to the end of the current/next WORD (space-delimited)
func (cv *ConversationView) MoveBigWordEnd() {
	if cv.cursorLine >= len(cv.renderedLines) {
		return
	}

	line := cv.renderedLines[cv.cursorLine].Content
	
	// Move forward at least one character
	if cv.cursorCol < len(line)-1 {
		cv.cursorCol++
	} else if cv.cursorLine < cv.totalLines-1 {
		cv.cursorLine++
		cv.cursorCol = 0
		if cv.cursorLine < len(cv.renderedLines) {
			line = cv.renderedLines[cv.cursorLine].Content
		}
	} else {
		return
	}

	// Skip whitespace
	for cv.cursorCol < len(line) && isWhitespace(rune(line[cv.cursorCol])) {
		cv.cursorCol++
	}

	// Move to end of non-whitespace
	for cv.cursorCol < len(line)-1 && !isWhitespace(rune(line[cv.cursorCol+1])) {
		cv.cursorCol++
	}

	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveToPreviousParagraph moves to the previous paragraph or message boundary
func (cv *ConversationView) MoveToPreviousParagraph() {
	// Move to previous empty line or message boundary
	startedOnEmpty := false
	if cv.cursorLine < len(cv.renderedLines) {
		startedOnEmpty = cv.renderedLines[cv.cursorLine].Content == ""
	}

	// If on empty line, skip to previous non-empty
	if startedOnEmpty {
		for cv.cursorLine > 0 && cv.renderedLines[cv.cursorLine].Content == "" {
			cv.cursorLine--
		}
	}

	// Skip non-empty lines until we find an empty line or message boundary
	currentMessageIdx := -1
	if cv.cursorLine < len(cv.renderedLines) {
		currentMessageIdx = cv.renderedLines[cv.cursorLine].MessageIndex
	}

	for cv.cursorLine > 0 {
		cv.cursorLine--
		if cv.renderedLines[cv.cursorLine].Content == "" {
			break
		}
		// Check for message boundary
		if cv.renderedLines[cv.cursorLine].MessageIndex != currentMessageIdx {
			break
		}
	}

	cv.cursorCol = 0
	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveToNextParagraph moves to the next paragraph or message boundary
func (cv *ConversationView) MoveToNextParagraph() {
	// Move to next empty line or message boundary
	startedOnEmpty := false
	if cv.cursorLine < len(cv.renderedLines) {
		startedOnEmpty = cv.renderedLines[cv.cursorLine].Content == ""
	}

	// If on empty line, skip to next non-empty
	if startedOnEmpty {
		for cv.cursorLine < cv.totalLines-1 && cv.renderedLines[cv.cursorLine].Content == "" {
			cv.cursorLine++
		}
	}

	// Skip non-empty lines until we find an empty line or message boundary
	currentMessageIdx := -1
	if cv.cursorLine < len(cv.renderedLines) {
		currentMessageIdx = cv.renderedLines[cv.cursorLine].MessageIndex
	}

	for cv.cursorLine < cv.totalLines-1 {
		cv.cursorLine++
		if cv.renderedLines[cv.cursorLine].Content == "" {
			break
		}
		// Check for message boundary
		if cv.renderedLines[cv.cursorLine].MessageIndex != currentMessageIdx {
			break
		}
	}

	cv.cursorCol = 0
	cv.updateViewport()
	cv.needsRedraw = true
}

// MoveToViewportTop moves cursor to the top of the visible viewport
func (cv *ConversationView) MoveToViewportTop() {
	cv.cursorLine = cv.viewportTop
	cv.cursorCol = 0
	cv.needsRedraw = true
}

// MoveToViewportMiddle moves cursor to the middle of the visible viewport
func (cv *ConversationView) MoveToViewportMiddle() {
	cv.cursorLine = cv.viewportTop + cv.height/2
	if cv.cursorLine >= cv.totalLines {
		cv.cursorLine = cv.totalLines - 1
	}
	cv.cursorCol = 0
	cv.needsRedraw = true
}

// MoveToViewportBottom moves cursor to the bottom of the visible viewport
func (cv *ConversationView) MoveToViewportBottom() {
	cv.cursorLine = cv.viewportTop + cv.height - 1
	if cv.cursorLine >= cv.totalLines {
		cv.cursorLine = cv.totalLines - 1
	}
	cv.cursorCol = 0
	cv.needsRedraw = true
}

// Selection methods

// StartCharSelection starts character selection mode
func (cv *ConversationView) StartCharSelection() {
	cv.SelectionMode = SelectionChar
	cv.selectionStart = Position{Line: cv.cursorLine, Col: cv.cursorCol}
	cv.selectionEnd = Position{Line: cv.cursorLine, Col: cv.cursorCol}
	cv.needsRedraw = true
}

// StartLineSelection starts line selection mode
func (cv *ConversationView) StartLineSelection() {
	cv.SelectionMode = SelectionLine
	cv.selectionStart = Position{Line: cv.cursorLine, Col: 0}
	cv.selectionEnd = Position{Line: cv.cursorLine, Col: 0}
	cv.needsRedraw = true
}

// StartBlockSelection starts block selection mode
func (cv *ConversationView) StartBlockSelection() {
	cv.SelectionMode = SelectionBlock
	cv.selectionStart = Position{Line: cv.cursorLine, Col: cv.cursorCol}
	cv.selectionEnd = Position{Line: cv.cursorLine, Col: cv.cursorCol}
	cv.needsRedraw = true
}

// CancelSelection cancels the current selection
func (cv *ConversationView) CancelSelection() {
	cv.SelectionMode = SelectionNone
	cv.needsRedraw = true
}

// UpdateSelection updates the selection end point based on cursor position
func (cv *ConversationView) UpdateSelection() {
	if cv.SelectionMode != SelectionNone {
		cv.selectionEnd = Position{Line: cv.cursorLine, Col: cv.cursorCol}
		cv.needsRedraw = true
	}
}

// GetSelectedText returns the currently selected text
func (cv *ConversationView) GetSelectedText() string {
	if cv.SelectionMode == SelectionNone {
		return ""
	}

	// Normalize selection bounds
	startLine := cv.selectionStart.Line
	endLine := cv.selectionEnd.Line
	startCol := cv.selectionStart.Col
	endCol := cv.selectionEnd.Col

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	var selected []string

	switch cv.SelectionMode {
	case SelectionChar:
		if startLine == endLine {
			// Single line selection
			if startLine < len(cv.renderedLines) {
				line := cv.renderedLines[startLine].Content
				if startCol < len(line) {
					end := endCol
					if end > len(line) {
						end = len(line)
					}
					selected = append(selected, line[startCol:end])
				}
			}
		} else {
			// Multi-line selection
			for i := startLine; i <= endLine && i < len(cv.renderedLines); i++ {
				line := cv.renderedLines[i].Content
				if i == startLine {
					if startCol < len(line) {
						selected = append(selected, line[startCol:])
					}
				} else if i == endLine {
					if endCol > 0 && endCol <= len(line) {
						selected = append(selected, line[:endCol])
					}
				} else {
					selected = append(selected, line)
				}
			}
		}

	case SelectionLine:
		for i := startLine; i <= endLine && i < len(cv.renderedLines); i++ {
			selected = append(selected, cv.renderedLines[i].Content)
		}

	case SelectionBlock:
		if startCol > endCol {
			startCol, endCol = endCol, startCol
		}
		for i := startLine; i <= endLine && i < len(cv.renderedLines); i++ {
			line := cv.renderedLines[i].Content
			if startCol < len(line) {
				end := endCol
				if end > len(line) {
					end = len(line)
				}
				selected = append(selected, line[startCol:end])
			}
		}
	}

	return strings.Join(selected, "\n")
}

// YankSelection copies the selection to clipboard
func (cv *ConversationView) YankSelection() error {
	text := cv.GetSelectedText()
	if text == "" {
		// If no selection, yank current line
		if cv.cursorLine < len(cv.renderedLines) {
			text = cv.renderedLines[cv.cursorLine].Content
		}
	}

	cv.yankBuffer = text

	// Copy to system clipboard
	if err := clipboard.WriteAll(text); err != nil {
		// Fallback to internal buffer only
		return fmt.Errorf("clipboard copy failed: %w", err)
	}

	cv.CancelSelection()
	return nil
}

// YankLine yanks the current line
func (cv *ConversationView) YankLine() error {
	if cv.cursorLine < len(cv.renderedLines) {
		text := cv.renderedLines[cv.cursorLine].Content
		cv.yankBuffer = text

		if err := clipboard.WriteAll(text); err != nil {
			return fmt.Errorf("clipboard copy failed: %w", err)
		}
	}
	return nil
}

// Render renders the conversation view
func (cv *ConversationView) Render() string {
	var output strings.Builder

	// Styles
	lineNumberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	userMsgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	assistantMsgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	selectionStyle := lipgloss.NewStyle().Background(lipgloss.Color("27")) // Blue selection

	// Render visible lines
	for i := 0; i < cv.height; i++ {
		lineIdx := cv.viewportTop + i
		if lineIdx >= cv.totalLines {
			// Empty line padding
			output.WriteString(strings.Repeat(" ", cv.width))
			if i < cv.height-1 {
				output.WriteString("\n")
			}
			continue
		}

		line := cv.renderedLines[lineIdx]
		
		// Render line number
		lineNum := fmt.Sprintf("%4d ", lineIdx+1)
		if lineIdx == cv.cursorLine {
			// Highlight line number for cursor line
			output.WriteString(lineNumberStyle.Copy().Background(lipgloss.Color("236")).Render(lineNum))
		} else {
			output.WriteString(lineNumberStyle.Render(lineNum))
		}

		// Determine base style for content
		baseStyle := lipgloss.NewStyle()
		if line.IsUserMsg {
			baseStyle = userMsgStyle
		} else if line.IsAssistant {
			baseStyle = assistantMsgStyle
		}
		if line.IsError {
			baseStyle = errorStyle
		}

		// Render content with selection/cursor highlighting
		content := line.Content
		if len(content) < cv.width-5 {
			// Pad content to full width for consistent highlighting
			content = content + strings.Repeat(" ", cv.width-5-len(content))
		}

		if cv.SelectionMode != SelectionNone && cv.isLineInSelection(lineIdx) {
			// Apply selection highlighting character by character
			renderedContent := cv.renderLineWithSelection(lineIdx, content, baseStyle, selectionStyle)
			output.WriteString(renderedContent)
		} else if lineIdx == cv.cursorLine {
			// Apply cursor line highlighting
			output.WriteString(baseStyle.Copy().Background(lipgloss.Color("236")).Render(content))
		} else {
			// Normal rendering
			output.WriteString(baseStyle.Render(content))
		}
		
		if i < cv.height-1 {
			output.WriteString("\n")
		}
	}

	// Add status line
	statusLine := cv.renderStatusLine()
	output.WriteString("\n")
	output.WriteString(statusLine)

	return output.String()
}

// renderLineWithSelection renders a line with selection highlighting
func (cv *ConversationView) renderLineWithSelection(lineIdx int, content string, baseStyle, selectionStyle lipgloss.Style) string {
	var result strings.Builder
	
	// Get selection bounds for this line
	startCol, endCol := cv.getSelectionBoundsForLine(lineIdx)
	
	for i, ch := range content {
		if i >= startCol && i < endCol {
			// Character is selected
			if lineIdx == cv.cursorLine {
				// Cursor line + selection
				result.WriteString(baseStyle.Copy().Background(lipgloss.Color("27")).Bold(true).Render(string(ch)))
			} else {
				// Just selection
				result.WriteString(baseStyle.Copy().Background(lipgloss.Color("27")).Render(string(ch)))
			}
		} else if lineIdx == cv.cursorLine {
			// Cursor line without selection
			result.WriteString(baseStyle.Copy().Background(lipgloss.Color("236")).Render(string(ch)))
		} else {
			// Normal character
			result.WriteString(baseStyle.Render(string(ch)))
		}
	}
	
	return result.String()
}

// getSelectionBoundsForLine returns the start and end column for selection on a given line
func (cv *ConversationView) getSelectionBoundsForLine(lineIdx int) (int, int) {
	if cv.SelectionMode == SelectionNone {
		return 0, 0
	}

	startLine := cv.selectionStart.Line
	endLine := cv.selectionEnd.Line
	startCol := cv.selectionStart.Col
	endCol := cv.selectionEnd.Col

	// Normalize selection (make sure start is before end)
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	switch cv.SelectionMode {
	case SelectionChar:
		if lineIdx < startLine || lineIdx > endLine {
			return 0, 0
		}
		if lineIdx == startLine && lineIdx == endLine {
			return startCol, endCol + 1 // Include end character
		}
		if lineIdx == startLine {
			return startCol, len(cv.renderedLines[lineIdx].Content)
		}
		if lineIdx == endLine {
			return 0, endCol + 1
		}
		// Middle line - select entire line
		return 0, len(cv.renderedLines[lineIdx].Content)
		
	case SelectionLine:
		if lineIdx >= startLine && lineIdx <= endLine {
			return 0, len(cv.renderedLines[lineIdx].Content)
		}
		return 0, 0
		
	case SelectionBlock:
		if lineIdx < startLine || lineIdx > endLine {
			return 0, 0
		}
		if startCol > endCol {
			startCol, endCol = endCol, startCol
		}
		return startCol, endCol + 1
	}
	
	return 0, 0
}

// isLineInSelection checks if a line has any selection
func (cv *ConversationView) isLineInSelection(lineIdx int) bool {
	if cv.SelectionMode == SelectionNone {
		return false
	}

	startLine := cv.selectionStart.Line
	endLine := cv.selectionEnd.Line

	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}

	return lineIdx >= startLine && lineIdx <= endLine
}


// renderStatusLine renders the status line at the bottom
func (cv *ConversationView) renderStatusLine() string {
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("250"))

	var mode string
	switch cv.SelectionMode {
	case SelectionChar:
		mode = " [VISUAL]"
	case SelectionLine:
		mode = " [VISUAL LINE]"
	case SelectionBlock:
		mode = " [VISUAL BLOCK]"
	default:
		mode = ""
	}

	position := fmt.Sprintf("[%d/%d]", cv.cursorLine+1, cv.totalLines)
	status := fmt.Sprintf("[CONVERSATION VIEW]%s %s", mode, position)

	// Pad to full width
	padding := cv.width - len(status)
	if padding > 0 {
		status += strings.Repeat(" ", padding)
	}

	return statusStyle.Render(status)
}

// IsActive returns whether the conversation view is active
func (cv *ConversationView) IsActive() bool {
	return true
}

// NeedsRedraw returns whether the view needs to be redrawn
func (cv *ConversationView) NeedsRedraw() bool {
	return cv.needsRedraw
}

// SetNeedsRedraw marks the view as needing a redraw
func (cv *ConversationView) SetNeedsRedraw(needs bool) {
	cv.needsRedraw = needs
}