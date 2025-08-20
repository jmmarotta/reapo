package session

import (
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"reapo/internal/tui/components"
)

// Session manages the dual storage of messages for UI display and API communication
type Session struct {
	ID         string                      // Unique session identifier
	UIMessages []components.Message        // Messages for TUI display
	Messages   []anthropic.MessageParam    // Messages for Anthropic API
	
	// Track relationships between UI and API messages
	MessageMap map[string][]int            // UI message ID -> API message indices
	
	// File state tracking
	FileStates map[string]*FileState       // Track file read/edit status
	
	// Token management
	TokenCount int                         // Current token usage
	MaxTokens  int                         // Maximum allowed tokens
	
	// Metadata
	StartedAt  time.Time
	UpdatedAt  time.Time
	
	mu         sync.RWMutex                // Thread safety
}

// FileState tracks the state of files accessed during the session
type FileState struct {
	Path         string
	LastRead     time.Time
	LastModified time.Time
	WasEdited    bool
	Content      string // Cache the content to detect changes
}

// NewSession creates a new session with default settings
func NewSession() *Session {
	return &Session{
		ID:         generateSessionID(),
		UIMessages: make([]components.Message, 0),
		Messages:   make([]anthropic.MessageParam, 0),
		MessageMap: make(map[string][]int),
		FileStates: make(map[string]*FileState),
		MaxTokens:  200000, // Default to 200k context window
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// AddUserAPIMessage adds a user message only to API messages (not UI)
// Smart: appends to last message if it's already a user message
func (s *Session) AddUserAPIMessage(text string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var apiMsgIndex int
	
	// Check if we should append to last message
	if len(s.Messages) > 0 {
		lastMsgIndex := len(s.Messages) - 1
		lastMsg := &s.Messages[lastMsgIndex]
		if lastMsg.Role == "user" {
			// Append as additional text block to existing user message
			lastMsg.Content = append(lastMsg.Content, anthropic.ContentBlockParamUnion{
				OfText: &anthropic.TextBlockParam{
					Type: "text",
					Text: text,
				},
			})
			apiMsgIndex = lastMsgIndex
		} else {
			// Create new user message (last message is not user)
			s.Messages = append(s.Messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(text),
			))
			apiMsgIndex = len(s.Messages) - 1
		}
	} else {
		// No messages yet, create first user message
		s.Messages = append(s.Messages, anthropic.NewUserMessage(
			anthropic.NewTextBlock(text),
		))
		apiMsgIndex = 0
	}
	
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
	return apiMsgIndex
}

// AddUserUIMessage adds a user message only to UI messages
func (s *Session) AddUserUIMessage(text string, messageID string, apiMsgIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add to UI messages
	uiMsg := components.Message{
		ID:        messageID,
		Role:      "user",
		Content:   text,
		Type:      components.MessageTypeText,
		Status:    components.MessageCompleted,
		Timestamp: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.UIMessages = append(s.UIMessages, uiMsg)
	
	// Track the relationship if API index provided
	if apiMsgIndex >= 0 {
		if s.MessageMap[messageID] == nil {
			s.MessageMap[messageID] = []int{}
		}
		s.MessageMap[messageID] = append(s.MessageMap[messageID], apiMsgIndex)
	}
}

// AddUserMessage adds a user message to both stores
func (s *Session) AddUserMessage(text string, messageID string) {
	// First add to API messages and get the index
	apiIndex := s.AddUserAPIMessage(text)
	// Then add to UI messages with the API index
	s.AddUserUIMessage(text, messageID, apiIndex)
}

// AddAssistantTextResponse adds an assistant text response to both stores
func (s *Session) AddAssistantTextResponse(text string, messageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add to UI messages
	uiMsg := components.Message{
		ID:        messageID,
		Role:      "assistant",
		Content:   text,
		Type:      components.MessageTypeText,
		Status:    components.MessageCompleted,
		Timestamp: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.UIMessages = append(s.UIMessages, uiMsg)
	
	// Add to API messages
	apiMsgIndex := len(s.Messages)
	s.Messages = append(s.Messages, anthropic.NewAssistantMessage(
		anthropic.NewTextBlock(text),
	))
	
	// Track the relationship
	s.MessageMap[messageID] = []int{apiMsgIndex}
	
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
}

// AddToolUse adds a tool use to the conversation
func (s *Session) AddToolUse(toolID string, toolName string, input interface{}, uiMessageID string, displayContent string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add formatted version to UI messages if display content is provided
	if displayContent != "" {
		uiMsg := components.Message{
			ID:        uiMessageID,
			Role:      "assistant",
			Content:   displayContent,
			Type:      components.MessageTypeText,
			Status:    components.MessageCompleted,
			Timestamp: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.UIMessages = append(s.UIMessages, uiMsg)
	}
	
	// Add to API messages as assistant message with tool use block
	apiMsgIndex := len(s.Messages)
	s.Messages = append(s.Messages, anthropic.NewAssistantMessage(
		anthropic.NewToolUseBlock(toolID, input, toolName),
	))
	
	// Track the relationship if we have a UI message
	if displayContent != "" {
		if s.MessageMap[uiMessageID] == nil {
			s.MessageMap[uiMessageID] = []int{}
		}
		s.MessageMap[uiMessageID] = append(s.MessageMap[uiMessageID], apiMsgIndex)
	}
	
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
}

// AddToolResult adds a tool result to the conversation
func (s *Session) AddToolResult(toolID string, result string, isError bool, shouldDisplay bool, uiMessageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add to UI messages if the output should be displayed
	if shouldDisplay && result != "" {
		status := components.MessageCompleted
		if isError {
			status = components.MessageError
		}
		
		uiMsg := components.Message{
			ID:        uiMessageID,
			Role:      "assistant",
			Content:   result,
			Type:      components.MessageTypeText,
			Status:    status,
			Timestamp: time.Now(),
			UpdatedAt: time.Now(),
			IsError:   isError,
		}
		s.UIMessages = append(s.UIMessages, uiMsg)
	}
	
	// Add to API messages as user message with tool result block
	apiMsgIndex := len(s.Messages)
	s.Messages = append(s.Messages, anthropic.NewUserMessage(
		anthropic.NewToolResultBlock(toolID, result, isError),
	))
	
	// Track the relationship if we have a UI message
	if shouldDisplay && uiMessageID != "" {
		if s.MessageMap[uiMessageID] == nil {
			s.MessageMap[uiMessageID] = []int{}
		}
		s.MessageMap[uiMessageID] = append(s.MessageMap[uiMessageID], apiMsgIndex)
	}
	
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
}

// AddToolUseAndResult adds both tool use and result in one operation (common pattern)
func (s *Session) AddToolUseAndResult(toolID string, toolName string, input interface{}, result string, isError bool, displayContent string, uiMessageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add UI message for the tool invocation with display content
	if displayContent != "" {
		uiMsg := components.Message{
			ID:        uiMessageID,
			Role:      "assistant",
			Content:   displayContent,
			Type:      components.MessageTypeText,
			Status:    components.MessageCompleted,
			Timestamp: time.Now(),
			UpdatedAt: time.Now(),
		}
		s.UIMessages = append(s.UIMessages, uiMsg)
	}
	
	// Add assistant message with tool use block
	toolUseIndex := len(s.Messages)
	s.Messages = append(s.Messages, anthropic.NewAssistantMessage(
		anthropic.NewToolUseBlock(toolID, input, toolName),
	))
	
	// Add user message with tool result block
	toolResultIndex := len(s.Messages)
	s.Messages = append(s.Messages, anthropic.NewUserMessage(
		anthropic.NewToolResultBlock(toolID, result, isError),
	))
	
	// Track both API message indices for this UI message
	if displayContent != "" && uiMessageID != "" {
		s.MessageMap[uiMessageID] = []int{toolUseIndex, toolResultIndex}
	}
	
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
}

// MarkFileAsRead tracks that a file has been read
func (s *Session) MarkFileAsRead(path string, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.FileStates[path] == nil {
		s.FileStates[path] = &FileState{
			Path: path,
		}
	}
	
	s.FileStates[path].LastRead = time.Now()
	s.FileStates[path].Content = content
}

// MarkFileAsEdited tracks that a file has been edited
func (s *Session) MarkFileAsEdited(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.FileStates[path] == nil {
		s.FileStates[path] = &FileState{
			Path: path,
		}
	}
	
	s.FileStates[path].LastModified = time.Now()
	s.FileStates[path].WasEdited = true
}

// IsFileRead checks if a file has been read in this session
func (s *Session) IsFileRead(path string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	state, exists := s.FileStates[path]
	return exists && !state.LastRead.IsZero()
}

// GetUIMessages returns a copy of UI messages for display
func (s *Session) GetUIMessages() []components.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	messages := make([]components.Message, len(s.UIMessages))
	copy(messages, s.UIMessages)
	return messages
}

// GetMessages returns a copy of API messages for sending to Anthropic
func (s *Session) GetMessages() []anthropic.MessageParam {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	messages := make([]anthropic.MessageParam, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}

// AppendMessage adds a single message to the API Messages
func (s *Session) AppendMessage(msg anthropic.MessageParam) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
}

// AppendMessages adds multiple messages to the API Messages
func (s *Session) AppendMessages(msgs []anthropic.MessageParam) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.Messages = append(s.Messages, msgs...)
	s.UpdatedAt = time.Now()
	s.updateTokenCount()
}

// GetTokenCount returns the current token count
func (s *Session) GetTokenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.TokenCount
}

// Clear resets the session
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.UIMessages = make([]components.Message, 0)
	s.Messages = make([]anthropic.MessageParam, 0)
	s.MessageMap = make(map[string][]int)
	s.FileStates = make(map[string]*FileState)
	s.TokenCount = 0
	s.UpdatedAt = time.Now()
}

// NewSessionFromCompaction creates a new session from a compacted conversation
func NewSessionFromCompaction(oldSession *Session, summary string, summaryMessageID string) *Session {
	oldSession.mu.RLock()
	defer oldSession.mu.RUnlock()
	
	// Create a new session with a new ID
	newSession := &Session{
		ID:         generateSessionID(),
		UIMessages: make([]components.Message, 0),
		Messages:   make([]anthropic.MessageParam, 0),
		MessageMap: make(map[string][]int),
		FileStates: make(map[string]*FileState),
		MaxTokens:  oldSession.MaxTokens, // Preserve max tokens setting
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	// Optionally preserve recent file states (files edited in last 5 minutes)
	cutoffTime := time.Now().Add(-5 * time.Minute)
	for path, state := range oldSession.FileStates {
		if state.WasEdited && state.LastModified.After(cutoffTime) {
			// Create a copy of the FileState
			newState := &FileState{
				Path:         state.Path,
				LastRead:     state.LastRead,
				LastModified: state.LastModified,
				WasEdited:    state.WasEdited,
				Content:      state.Content,
			}
			newSession.FileStates[path] = newState
		}
	}
	
	// Add the summary as the first message
	if summary != "" {
		// Add to UI messages
		uiMsg := components.Message{
			ID:        summaryMessageID,
			Role:      "user",
			Content:   summary,
			Type:      components.MessageTypeText,
			Status:    components.MessageCompleted,
			Timestamp: time.Now(),
			UpdatedAt: time.Now(),
		}
		newSession.UIMessages = append(newSession.UIMessages, uiMsg)
		
		// Add to API messages
		newSession.Messages = append(newSession.Messages, anthropic.NewUserMessage(
			anthropic.NewTextBlock(summary),
		))
		
		// Track the relationship
		newSession.MessageMap[summaryMessageID] = []int{0}
	}
	
	// Set initial token count
	newSession.TokenCount = len(summary) / 4 // Rough estimate
	
	return newSession
}

// updateTokenCount updates the token count (simplified - should use proper tokenizer)
func (s *Session) updateTokenCount() {
	// Simple approximation: 1 token ≈ 4 characters
	// In production, use proper tokenizer
	totalChars := 0
	
	for _, msg := range s.Messages {
		// This is a simplified count - actual implementation would
		// properly serialize and count the message content
		// For now, approximate based on string length
		totalChars += estimateMessageChars(msg)
	}
	
	s.TokenCount = totalChars / 4
}

// estimateMessageChars estimates character count for a message (simplified)
func estimateMessageChars(msg anthropic.MessageParam) int {
	// This is a placeholder - actual implementation would
	// properly count all content blocks
	// For now, return a rough estimate
	return 100 // Placeholder
}

// HasAPIMessages checks if the session has any API messages
func (s *Session) HasAPIMessages() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return len(s.Messages) > 0
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return time.Now().Format("20060102-150405")
}