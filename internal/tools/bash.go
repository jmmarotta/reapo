package tools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"reapo/internal/schema"
)

//go:embed prompts/bash.txt
var bashPrompt string

// BashDefinition tool definition
var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: bashPrompt,
	InputSchema: schema.GenerateSchema[BashInput](),
	Function:    Bash,
}

type BashInput struct {
	Command     string `json:"command" jsonschema_description:"The command to execute"`
	Description string `json:"description,omitempty" jsonschema_description:"Clear, concise description of what this command does in 5-10 words. Examples:\nInput: ls\nOutput: Lists files in current directory\n\nInput: git status\nOutput: Shows working tree status\n\nInput: npm install\nOutput: Installs package dependencies\n\nInput: mkdir foo\nOutput: Creates directory 'foo'"`
	Timeout     int    `json:"timeout,omitempty" jsonschema_description:"Optional timeout in milliseconds (max 600000)"`
}

func Bash(input json.RawMessage) (string, error) {
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if bashInput.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Set default timeout if not specified
	timeout := 120000 // 2 minutes default
	if bashInput.Timeout > 0 {
		if bashInput.Timeout > 600000 {
			timeout = 600000 // Max 10 minutes
		} else {
			timeout = bashInput.Timeout
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Execute command
	cmd := exec.CommandContext(ctx, "bash", "-c", bashInput.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.String() != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Truncate output if too long
	const maxOutput = 30000
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n\n[Output truncated to 30000 characters]"
	}

	// Handle errors
	if ctx.Err() == context.DeadlineExceeded {
		return output + fmt.Sprintf("\n\n[Command timed out after %dms]", timeout), fmt.Errorf("command timed out after %dms", timeout)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Command executed but returned non-zero exit code
			return output, fmt.Errorf("command exited with code %d", exitErr.ExitCode())
		}
		// Other execution errors
		return output, fmt.Errorf("failed to execute command: %w", err)
	}

	return strings.TrimSpace(output), nil
}
