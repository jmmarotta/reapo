package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"reapo/internal/schema"
)

// MultiEditDefinition tool definition
var MultiEditDefinition = ToolDefinition{
	Name: "multiedit",
	Description: `This is a tool for making multiple edits to a single file in one operation. It is built on top of the Edit tool and allows you to perform multiple find-and-replace operations efficiently. Prefer this tool over the Edit tool when you need to make multiple edits to the same file.

Before using this tool:

1. Use the Read tool to understand the file's contents and context
2. Verify the directory path is correct

To make multiple file edits, provide the following:
1. file_path: The absolute path to the file to modify (must be absolute, not relative)
2. edits: An array of edit operations to perform, where each edit contains:
   - old_string: The text to replace (must match the file contents exactly, including all whitespace and indentation)
   - new_string: The edited text to replace the old_string
   - replace_all: Replace all occurences of old_string. This parameter is optional and defaults to false.

IMPORTANT:
- All edits are applied in sequence, in the order they are provided
- Each edit operates on the result of the previous edit
- All edits must be valid for the operation to succeed - if any edit fails, none will be applied
- This tool is ideal when you need to make several changes to different parts of the same file

CRITICAL REQUIREMENTS:
1. All edits follow the same requirements as the single Edit tool
2. The edits are atomic - either all succeed or none are applied
3. Plan your edits carefully to avoid conflicts between sequential operations

WARNING:
- The tool will fail if edits.old_string doesn't match the file contents exactly (including whitespace)
- The tool will fail if edits.old_string and edits.new_string are the same
- Since edits are applied in sequence, ensure that earlier edits don't affect the text that later edits are trying to find

When making edits:
- Ensure all edits result in idiomatic, correct code
- Do not leave the code in a broken state
- Always use absolute file paths (starting with /)
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- Use replace_all for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.

If you want to create a new file, use:
- A new file path, including dir name if needed
- First edit: empty old_string and the new file's contents as new_string
- Subsequent edits: normal edit operations on the created content`,
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
