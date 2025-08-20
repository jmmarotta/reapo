package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"reapo/internal/agent"
	"reapo/internal/config"
	"reapo/internal/session"
	"reapo/internal/tools"
	"reapo/internal/tui/completion"
	"reapo/internal/tui/components"
	"reapo/internal/tui/components/vimtextarea"
)

// FocusedWindow represents which window is currently focused
type FocusedWindow int

const (
	FocusInput FocusedWindow = iota
	FocusChat
)

// Model represents the Bubble Tea model for the TUI
type Model struct {
	session  *session.Session // Manages dual message storage
	textarea vimtextarea.Model
	viewport struct {
		width  int
		height int
	}
	agent             *agent.Agent
	client            anthropic.Client
	toolDefs          []tools.ToolDefinition
	ready             bool
	processing        bool
	processingText    string // Text to show during processing
	processingSpinner *components.SpinnerComponent
	maxContextTokens  int                                     // Maximum context window size (200k for both models)
	currentModel      string                                  // Current model being used
	spinners          map[string]*components.SpinnerComponent // Track spinners by message ID
	helpModal         *components.HelpModal                   // Help modal
	statusModal       *components.StatusModal                 // Status modal
	statusline        *components.StatuslineComponent         // Statusline for messages
	// Auth state
	authVerifier string // OAuth verifier for code exchange
	authModal    components.AuthModal
	// Conversation view state
	conversationViewMode bool                         // Whether conversation view is active
	conversationView     *components.ConversationView // The conversation view component
	leaderKey           string                       // The configured leader key
	lastKeyWasLeader    bool                         // Track if last key was leader
	lastKeyWasG         bool                         // Track if last key was 'g' for gg command
	// Window focus and scrolling
	focusedWindow         FocusedWindow // Which window is currently focused
	chatScrollOffset      int           // Lines from bottom (0 = viewing latest)
	chatTotalLines        int           // Total number of rendered chat lines
	chatCursorLine        int           // Cursor position in chat viewport
	awaitingWindowCommand bool          // Waiting for window navigation command after Ctrl-W
	// Visual selection state for chat
	chatVisualMode        bool // Whether visual mode is active in chat
	chatVisualLine        bool // true for line mode (V), false for char mode (v)
	chatSelectionStart    int  // Starting line of selection
	chatSelectionEnd      int  // Ending line of selection
}

// AgentResponseMsg represents a message from the agent
type AgentResponseMsg struct {
	Content string
	IsError bool
}

// AddMessageMsg represents adding a new message to the chat
type AddMessageMsg struct {
	Message components.Message
}

// MessageUpdateMsg represents updating an existing message
type MessageUpdateMsg struct {
	MessageID string
	Content   string
	Status    components.MessageStatus
	Progress  *components.Progress
	ToolInfo  *components.ToolInfo
}

// ToolInvocationMsg represents a tool being invoked
type ToolInvocationMsg struct {
	ToolName  string
	ToolID    string
	Input     string
	MessageID string
}

// ToolResultMsg represents a tool execution result
type ToolResultMsg struct {
	ToolName  string
	ToolID    string
	Output    string
	Error     string
	Duration  string
	MessageID string
}

// AnimationTickMsg represents a tick for spinner animations
type AnimationTickMsg struct{}

// ProcessMessageSequenceMsg represents the start of message processing sequence
type ProcessMessageSequenceMsg struct {
	UserMessage    string
	UserMessageID  string
	AgentMessageID string
}

// AgentStatusMsg represents agent thinking/status updates
type AgentStatusMsg struct {
	Message   string
	Timestamp time.Time
}

// ProcessToolsMsg triggers processing of tool uses
type ProcessToolsMsg struct {
	Response       *anthropic.Message
	AgentMessageID string
}

// SlashCommandMsg represents a slash command to be executed
type SlashCommandMsg struct {
	Command string
}

// ShowHelpModalMsg triggers showing the help modal
type ShowHelpModalMsg struct{}

// ShowStatusModalMsg triggers showing the status modal
type ShowStatusModalMsg struct{}

// ClearConversationMsg triggers clearing the conversation
type ClearConversationMsg struct{}

// OpenExternalEditorMsg triggers opening the external editor
type OpenExternalEditorMsg struct{}

// CompactConversationMsg triggers conversation compaction with the summary result
type CompactConversationMsg struct {
	Summary string
	Error   error
	IsAuto  bool // Whether this is an automatic compaction
}

// ShowStatuslineMsg displays a message in the statusline
type ShowStatuslineMsg struct {
	Type     components.StatuslineMessageType
	Text     string
	Duration time.Duration
}

// ClearStatuslineMsg clears the statusline message
type ClearStatuslineMsg struct{}

// systemMessages will be set by the runner
var systemMessages []anthropic.TextBlockParam

// NewModel creates a new TUI model
func NewModel(client anthropic.Client, toolDefs []tools.ToolDefinition) Model {
	// Initialize vim textarea
	ta := vimtextarea.New()
	ta.SetPlaceholder("Type a message... (Enter in Normal, Ctrl+S in Insert/Visual)")
	ta.Focus()
	ta.SetHeight(1)

	// Get working directory for completion engine
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "." // fallback to current directory
	}

	// Initialize completion engine
	completionEngine := completion.NewCompletionEngine(workingDir)
	ta.SetCompletionEngine(completionEngine)

	chatAgent := agent.NewAgent(&client, nil, toolDefs, systemMessages)

	// Initialize session
	sess := session.NewSession()
	sess.MaxTokens = config.GetContextTokens()

	model := Model{
		session:              sess,
		textarea:             ta,
		agent:                chatAgent,
		client:                client,
		toolDefs:              toolDefs,
		maxContextTokens:      config.GetContextTokens(),
		currentModel:          config.GetModelName(),
		spinners:              make(map[string]*components.SpinnerComponent),
		helpModal:             components.NewHelpModal(),
		statusModal:           components.NewStatusModal(),
		statusline:            components.NewStatuslineComponent(0), // Width will be set on WindowSizeMsg
		authModal:             components.NewAuthModal(),
		leaderKey:             config.GetLeaderKey(),
		conversationViewMode:  false,
		conversationView:      nil,
		lastKeyWasLeader:      false,
		lastKeyWasG:           false,
		focusedWindow:         FocusInput, // Start with input focused
		chatScrollOffset:      0,           // Start at bottom of chat
		chatTotalLines:        0,
		chatCursorLine:        0,
		awaitingWindowCommand: false,
		chatVisualMode:        false,
		chatVisualLine:        false,
		chatSelectionStart:    0,
		chatSelectionEnd:      0,
	}

	return model
}

// Init initializes the TUI model
func (m *Model) Init() tea.Cmd {
	return m.textarea.Init()
}

// generateMessageID creates a unique UUIDv7-based message ID
func generateMessageID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to a timestamp-based ID if UUID generation fails
		return fmt.Sprintf("msg_fallback_%d", time.Now().UnixNano())
	}
	return id.String()
}

// calculateChatHeight calculates the available height for the chat component
// This is a simplified version that estimates based on known component heights
func (m *Model) calculateChatHeight() int {
	// Estimate component heights
	textareaHeight := m.textarea.Height()
	inputBorderHeight := 2 // Border adds 2 lines
	inputMargins := 4      // MarginTop(1) + MarginBottom(3) = 4 lines
	footerHeight := 1      // Footer is 1 line
	statuslineHeight := 1  // Statusline is 1 line
	
	// No fixed spacing needed - lipgloss.JoinVertical handles layout
	fixedSpacing := 0
	
	// Processing indicator adds 1 line when active  
	processingHeight := 0
	if m.processing {
		processingHeight = 1
	}
	
	// Completion height when active
	completionHeight := 0
	if completionState := m.textarea.CompletionState(); completionState.Active {
		// Estimate completion height (usually 5-10 lines)
		completionHeight = min(10, len(completionState.Items))
	}
	
	// Total height of everything except chat
	nonChatHeight := textareaHeight + inputBorderHeight + inputMargins + 
		footerHeight + statuslineHeight + fixedSpacing + 
		processingHeight + completionHeight
	
	// Chat fills the remaining space
	chatHeight := m.viewport.height - nonChatHeight
	
	// Ensure we have at least 1 line for chat
	if chatHeight < 1 {
		chatHeight = 1
	}
	
	return chatHeight
}
