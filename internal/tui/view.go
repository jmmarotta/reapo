package tui

import (
	"github.com/charmbracelet/lipgloss"
	"reapo/internal/tui/components"
)

// View renders the TUI
func (m *Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Render conversation view if active (overlay)
	if m.conversationViewMode && m.conversationView != nil {
		return m.conversationView.Render()
	}

	// Get completion state
	completionState := m.textarea.CompletionState()

	// Create completion component if active
	var completionComponent components.CompletionComponent
	var completion string
	var completionHeight int
	if completionState.Active {
		completionComponent = components.NewCompletionComponent(
			completionState.Items,
			completionState.Selected,
			m.viewport.width,
		)
		completion = completionComponent.Render()
		completionHeight = lipgloss.Height(completion)
	}

	// Render processing indicator if active
	var processingIndicator string
	var processingHeight int
	if m.processing && m.processingSpinner != nil {
		processingIndicator = "  " + m.processingSpinner.View() + " " + m.processingText
		processingHeight = lipgloss.Height(processingIndicator)
	}

	// Render input component
	inputComponent := components.NewInputComponent(m.textarea, m.viewport.width)
	input := inputComponent.Render()
	inputHeight := lipgloss.Height(input)

	// Render footer component
	footerComponent := components.NewFooterComponent(m.textarea.Mode(), m.viewport.width)
	footerComponent.UpdateContextInfo(m.session.GetTokenCount(), m.maxContextTokens, m.currentModel)
	// Set focused window for footer display
	if m.focusedWindow == FocusChat {
		footerComponent.SetFocusedWindow("chat")
		// Calculate actual cursor position in chat (accounting for scroll)
		// We'll update this after calculating chatHeight
	} else {
		footerComponent.SetFocusedWindow("input")
	}
	footer := footerComponent.Render()
	footerHeight := lipgloss.Height(footer)

	// Render statusline
	statusline := ""
	statuslineHeight := 0
	if m.statusline != nil {
		statusline = m.statusline.Render()
		statuslineHeight = lipgloss.Height(statusline)
	}

	// Calculate available height for chat
	// Layout: chat + processingIndicator + completion + input + footer + "\n" + statusline
	// Fixed spacing: 0 (all spacing is now handled by component margins)
	fixedSpacing := 0

	// Calculate total height of everything except chat
	nonChatHeight := inputHeight + footerHeight + statuslineHeight + completionHeight + processingHeight + fixedSpacing

	// Chat should fill exactly the remaining space
	chatHeight := m.viewport.height - nonChatHeight

	// Ensure we have at least 1 line for chat
	if chatHeight < 1 {
		chatHeight = 1
	}

	// Now update footer with cursor info if in chat mode
	if m.focusedWindow == FocusChat {
		// Calculate actual line position
		var startIdx int
		if m.chatTotalLines <= chatHeight {
			startIdx = 0
		} else {
			endIdx := m.chatTotalLines - m.chatScrollOffset
			startIdx = max(0, endIdx - chatHeight)
		}
		actualLine := startIdx + m.chatCursorLine
		footerComponent.SetChatCursorInfo(actualLine, m.chatTotalLines)
		footer = footerComponent.Render()
	}

	// Create and render chat component with calculated height
	chatComponent := components.NewChatComponentWithSelection(
		m.session.GetUIMessages(),
		chatHeight,
		m.viewport.width,
		m.chatScrollOffset,
		m.focusedWindow == FocusChat,
		m.chatCursorLine,
		m.chatVisualMode,
		m.chatVisualLine,
		m.chatSelectionStart,
		m.chatSelectionEnd,
	)
	chat, totalLines := chatComponent.RenderWithSpinners(m.spinners)
	// Store the actual line count for accurate scrolling
	m.chatTotalLines = totalLines

	// Render help modal if visible (overlay on top)
	if m.helpModal.IsVisible() {
		return m.helpModal.View()
	}

	// Render status modal if visible (overlay on top)
	if m.statusModal.IsVisible() {
		return m.statusModal.View()
	}

	// Render auth modal if active (overlay on top)
	if m.authModal.Active() {
		return m.authModal.View()
	}

	// Build the layout components
	var layoutParts []string
	
	// Add chat (always present)
	layoutParts = append(layoutParts, chat)
	
	// Add optional components
	if processingIndicator != "" {
		layoutParts = append(layoutParts, processingIndicator)
	}
	if completion != "" {
		layoutParts = append(layoutParts, completion)
	}
	
	// Add required components
	layoutParts = append(layoutParts, input)
	layoutParts = append(layoutParts, footer)
	if statusline != "" {
		layoutParts = append(layoutParts, statusline)
	}
	
	// Join all parts vertically
	return lipgloss.JoinVertical(lipgloss.Left, layoutParts...)
}
