package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"reapo/internal/schema"
)

//go:embed prompts/write.txt
var writePrompt string

// WriteDefinition tool definition
var WriteDefinition = ToolDefinition{
	Name:        "write",
	Description: writePrompt,
	InputSchema: schema.GenerateSchema[WriteInput](),
	Function:    Write,
}

type WriteInput struct {
	FilePath string `json:"file_path" jsonschema_description:"The absolute path to the file to write (must be absolute, not relative)"`
	Content  string `json:"content" jsonschema_description:"The content to write to the file"`
}

// Track files that have been read to enforce the read-before-write rule
var readFiles = make(map[string]bool)

// MarkFileAsRead marks a file as having been read (called by Read tool)
func MarkFileAsRead(filePath string) {
	absPath, _ := filepath.Abs(filePath)
	readFiles[absPath] = true
}

func Write(input json.RawMessage) (string, error) {
	writeInput := WriteInput{}
	err := json.Unmarshal(input, &writeInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if writeInput.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	if !filepath.IsAbs(writeInput.FilePath) {
		return "", fmt.Errorf("file_path must be an absolute path, not relative")
	}

	// Check if file exists
	fileInfo, err := os.Stat(writeInput.FilePath)
	fileExists := err == nil && !fileInfo.IsDir()

	// If file exists, check if it was read first
	if fileExists {
		absPath, _ := filepath.Abs(writeInput.FilePath)
		if !readFiles[absPath] {
			return "", fmt.Errorf("you must use the Read tool to read the file contents before overwriting an existing file")
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(writeInput.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	err = os.WriteFile(writeInput.FilePath, []byte(writeInput.Content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if fileExists {
		return fmt.Sprintf("Successfully overwrote file: %s", writeInput.FilePath), nil
	}
	return fmt.Sprintf("Successfully created file: %s", writeInput.FilePath), nil
}
