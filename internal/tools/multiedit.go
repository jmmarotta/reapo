package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"reapo/internal/schema"
)

//go:embed prompts/multiedit.txt
var multiEditPrompt string

// MultiEditDefinition tool definition
var MultiEditDefinition = ToolDefinition{
	Name:        "multiedit",
	Description: multiEditPrompt,
	InputSchema: schema.GenerateSchema[MultiEditInput](),
	Function:    MultiEdit,
}

type MultiEditInput struct {
	FilePath string          `json:"file_path" jsonschema_description:"The absolute path to the file to modify"`
	Edits    []EditOperation `json:"edits" jsonschema:"minItems=1" jsonschema_description:"Array of edit operations to perform sequentially on the file"`
}

type EditOperation struct {
	OldString  string `json:"old_string" jsonschema_description:"The text to replace"`
	NewString  string `json:"new_string" jsonschema_description:"The text to replace it with"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"default=false" jsonschema_description:"Replace all occurences of old_string (default false)."`
}

func MultiEdit(input json.RawMessage) (string, error) {
	multiEditInput := MultiEditInput{}
	err := json.Unmarshal(input, &multiEditInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if multiEditInput.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	if !filepath.IsAbs(multiEditInput.FilePath) {
		return "", fmt.Errorf("file_path must be an absolute path, not relative")
	}

	if len(multiEditInput.Edits) == 0 {
		return "", fmt.Errorf("at least one edit is required")
	}

	// Read the file content
	content, err := os.ReadFile(multiEditInput.FilePath)
	fileExists := err == nil

	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// If file doesn't exist and first edit has empty old_string, create the file
	if !fileExists && multiEditInput.Edits[0].OldString == "" {
		// Create directory if needed
		dir := filepath.Dir(multiEditInput.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		content = []byte{}
	} else if !fileExists {
		return "", fmt.Errorf("file not found: %s", multiEditInput.FilePath)
	}

	// Apply edits sequentially
	currentContent := string(content)
	editCount := 0

	for i, edit := range multiEditInput.Edits {
		if edit.OldString == edit.NewString {
			return "", fmt.Errorf("edit %d: old_string and new_string must be different", i+1)
		}

		if edit.ReplaceAll {
			// Replace all occurrences
			newContent := strings.ReplaceAll(currentContent, edit.OldString, edit.NewString)
			if newContent == currentContent && edit.OldString != "" {
				return "", fmt.Errorf("edit %d: old_string not found in file", i+1)
			}
			currentContent = newContent
			editCount++
		} else {
			// Replace first occurrence only
			count := strings.Count(currentContent, edit.OldString)
			if count == 0 && edit.OldString != "" {
				return "", fmt.Errorf("edit %d: old_string not found in file", i+1)
			}
			if count > 1 {
				return "", fmt.Errorf("edit %d: old_string appears %d times in file. Use replace_all: true to replace all occurrences, or make old_string more specific", i+1, count)
			}
			currentContent = strings.Replace(currentContent, edit.OldString, edit.NewString, 1)
			editCount++
		}
	}

	// Write the modified content back
	err = os.WriteFile(multiEditInput.FilePath, []byte(currentContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully applied %d edit(s) to %s", editCount, multiEditInput.FilePath), nil
}
