package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"reapo/internal/agent"
	"reapo/internal/schema"
)

// Global variables to store client and system prompt for task execution
var (
	taskClient       *anthropic.Client
	taskSystemPrompt string
)

// InitializeTaskAgent sets up the global client and system prompt for task execution
func InitializeTaskAgent(client *anthropic.Client, systemPrompt string) {
	taskClient = client
	taskSystemPrompt = systemPrompt
}

type TaskInput struct {
	Description  string `json:"description" jsonschema_description:"A short (3-5 word) description of the task"`
	Prompt       string `json:"prompt" jsonschema_description:"The task for the agent to perform"`
	SubagentType string `json:"subagent_type" jsonschema_description:"The type of specialized agent to use for this task"`
}

// Task uses GenerateText to execute tasks
func Task(input json.RawMessage) (string, error) {
	if taskClient == nil {
		return "", fmt.Errorf("task client not initialized - call InitializeTaskAgent first")
	}

	var taskInput TaskInput
	if err := json.Unmarshal(input, &taskInput); err != nil {
		return "", fmt.Errorf("invalid task input: %w", err)
	}

	// Validate required fields
	if taskInput.Description == "" {
		return "", fmt.Errorf("description is required")
	}
	if taskInput.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if taskInput.SubagentType == "" {
		return "", fmt.Errorf("subagent_type is required")
	}

	// For now, we only support general-purpose agent
	if taskInput.SubagentType != "general-purpose" {
		return "", fmt.Errorf("unsupported subagent_type: %s (only 'general-purpose' is currently supported)", taskInput.SubagentType)
	}

	// Create an agent with all available tools for task execution
	availableTools := []agent.ToolDefinition{
		ReadDefinition,
		WriteDefinition,
		EditDefinition,
		MultiEditDefinition,
		LSDefinition,
		GlobDefinition,
		GrepDefinition,
		BashDefinition,
		WebFetchDefinition,
		WebSearchDefinition,
		TodoReadDefinition,
		TodoWriteDefinition,
		// Note: We don't include TaskDefinition to avoid recursion
	}

	taskAgent := agent.NewAgent(taskClient, nil, availableTools, taskSystemPrompt)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 minutes for complex tasks
	defer cancel()

	// Use GenerateText to execute the task
	return taskAgent.GenerateText(ctx, taskInput.Prompt)
}

//go:embed prompts/task.txt
var taskPrompt string

// TaskDefinition tool definition (renamed from RunTask for consistency)
var TaskDefinition = ToolDefinition{
	Name:        "task",
	Description: taskPrompt,
	InputSchema: schema.GenerateSchema[TaskInput](),
	Function:    Task,
}

// Keep RunTaskDefinition for backward compatibility
var RunTaskDefinition = TaskDefinition
