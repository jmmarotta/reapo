package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"reapo/internal/schema"
)

// ReadFileDefinition tool definition (renamed from ReadFile for consistency)
var ReadDefinition = ToolDefinition{
	Name: "read",
	Description: `Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage:
- The file_path parameter must be an absolute path, not a relative path
- By default, it reads up to 2000 lines starting from the beginning of the file
- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters
- Any lines longer than 2000 characters will be truncated
- Results are returned using cat -n format, with line numbers starting at 1
- This tool allows reapo to read images (eg PNG, JPG, etc). When reading an image file the contents are presented visually as reapo is a multimodal LLM.
- You have the capability to call multiple tools in a single response. It is always better to speculatively read multiple files as a batch that are potentially useful. 
- You will regularly be asked to read screenshots. If the user provides a path to a screenshot ALWAYS use this tool to view the file at the path. This tool will work with all temporary file paths like /var/folders/123/abc/T/TemporaryItems/NSIRD_screencaptureui_ZfB1tD/Screenshot.png
- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents.`,
	InputSchema: schema.GenerateSchema[ReadInput](),
	Function:    Read,
}

type ReadInput struct {
	FilePath string `json:"file_path" jsonschema_description:"The absolute path to the file to read"`
	Offset   int    `json:"offset,omitempty" jsonschema_description:"The line number to start reading from"`
	Limit    int    `json:"limit,omitempty" jsonschema_description:"The number of lines to read"`
}

const (
	DEFAULT_READ_LIMIT = 2000
	MAX_LINE_LENGTH    = 2000
)

func Read(input json.RawMessage) (string, error) {
	readInput := ReadInput{}
	err := json.Unmarshal(input, &readInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if readInput.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	if !filepath.IsAbs(readInput.FilePath) {
		return "", fmt.Errorf("file_path must be an absolute path, not relative")
	}

	// Check if file exists
	info, err := os.Stat(readInput.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Provide helpful suggestions
			dir := filepath.Dir(readInput.FilePath)
			base := filepath.Base(readInput.FilePath)

			entries, dirErr := os.ReadDir(dir)
			if dirErr == nil {
				var suggestions []string
				for _, entry := range entries {
					name := entry.Name()
					if strings.Contains(strings.ToLower(name), strings.ToLower(base)) ||
						strings.Contains(strings.ToLower(base), strings.ToLower(name)) {
						suggestions = append(suggestions, filepath.Join(dir, name))
						if len(suggestions) >= 3 {
							break
						}
					}
				}
				if len(suggestions) > 0 {
					return "", fmt.Errorf("file not found: %s\n\nDid you mean one of these?\n%s", readInput.FilePath, strings.Join(suggestions, "\n"))
				}
			}
			return "", fmt.Errorf("file not found: %s", readInput.FilePath)
		}
		return "", err
	}

	// Check if it's a directory
	if info.IsDir() {
		return "", fmt.Errorf("cannot read directory: %s (use LS tool instead)", readInput.FilePath)
	}

	// Check if it's an image
	if imageType := isImageFile(readInput.FilePath); imageType != "" {
		return "", fmt.Errorf("this is an image file of type: %s\nUse a different tool to process images", imageType)
	}

	// Read file content
	content, err := os.ReadFile(readInput.FilePath)
	if err != nil {
		return "", err
	}

	// Check if binary
	if isBinaryContent(content) {
		return "", fmt.Errorf("cannot read binary file: %s", readInput.FilePath)
	}

	// Mark file as read for the Write tool
	MarkFileAsRead(readInput.FilePath)

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Apply offset and limit
	offset := readInput.Offset
	if offset < 0 {
		offset = 0
	}
	limit := readInput.Limit
	if limit <= 0 {
		limit = DEFAULT_READ_LIMIT
	}

	// Get the requested lines
	endLine := offset + limit
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if offset > len(lines) {
		offset = len(lines)
	}

	selectedLines := lines[offset:endLine]

	// Format output with line numbers
	var output strings.Builder
	output.WriteString("<file>\n")

	for i, line := range selectedLines {
		lineNum := offset + i + 1
		// Truncate long lines
		if len(line) > MAX_LINE_LENGTH {
			line = line[:MAX_LINE_LENGTH] + "..."
		}
		// Format with 5-digit padded line numbers
		output.WriteString(fmt.Sprintf("%05d| %s\n", lineNum, line))
	}

	if endLine < len(lines) {
		output.WriteString(fmt.Sprintf("\n(File has more lines. Use 'offset' parameter to read beyond line %d)", endLine))
	}
	output.WriteString("\n</file>")

	return output.String(), nil
}

func isImageFile(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	imageTypes := map[string]string{
		".jpg":  "JPEG",
		".jpeg": "JPEG",
		".png":  "PNG",
		".gif":  "GIF",
		".bmp":  "BMP",
		".svg":  "SVG",
		".webp": "WebP",
	}
	return imageTypes[ext]
}

func isBinaryContent(content []byte) bool {
	// Check first 512 bytes for null bytes
	length := len(content)
	if length > 512 {
		length = 512
	}
	for i := 0; i < length; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// ListFiles tool definition
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	InputSchema: schema.GenerateSchema[ListFilesInput](),
	Function:    ListFiles,
}

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// EditDefinition tool definition (renamed from EditFile for consistency)
var EditDefinition = ToolDefinition{
	Name: "edit",
	Description: `Performs exact string replacements in files. 

Usage:
- You must use your ` + "`Read`" + ` tool at least once in the conversation before editing. This tool will error if you attempt an edit without reading the file. 
- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. The line number prefix format is: spaces + line number + tab. Everything after that tab is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- The edit will FAIL if ` + "`old_string`" + ` is not unique in the file. Either provide a larger string with more surrounding context to make it unique or use ` + "`replace_all`" + ` to change every instance of ` + "`old_string`" + `. 
- Use ` + "`replace_all`" + ` for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.`,
	InputSchema: schema.GenerateSchema[EditInput](),
	Function:    Edit,
}

type EditInput struct {
	FilePath   string `json:"file_path" jsonschema_description:"The absolute path to the file to modify"`
	OldString  string `json:"old_string" jsonschema_description:"The text to replace"`
	NewString  string `json:"new_string" jsonschema_description:"The text to replace it with (must be different from old_string)"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"default=false" jsonschema_description:"Replace all occurences of old_string (default false)"`
}

func Edit(input json.RawMessage) (string, error) {
	editInput := EditInput{}
	err := json.Unmarshal(input, &editInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if editInput.FilePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	if !filepath.IsAbs(editInput.FilePath) {
		return "", fmt.Errorf("file_path must be an absolute path, not relative")
	}

	if editInput.OldString == editInput.NewString {
		return "", fmt.Errorf("old_string and new_string must be different")
	}

	// Check if file has been read
	absPath, _ := filepath.Abs(editInput.FilePath)
	if _, wasRead := readFiles[absPath]; !wasRead {
		// Check if file exists
		if _, err := os.Stat(editInput.FilePath); err == nil {
			return "", fmt.Errorf("you must use the Read tool to read the file contents before editing")
		}
	}

	content, err := os.ReadFile(editInput.FilePath)
	if err != nil {
		if os.IsNotExist(err) && editInput.OldString == "" {
			// Create new file
			dir := filepath.Dir(editInput.FilePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
			err := os.WriteFile(editInput.FilePath, []byte(editInput.NewString), 0644)
			if err != nil {
				return "", fmt.Errorf("failed to create file: %w", err)
			}
			return fmt.Sprintf("Successfully created file %s", editInput.FilePath), nil
		}
		return "", err
	}

	oldContent := string(content)
	var newContent string

	if editInput.ReplaceAll {
		// Replace all occurrences
		newContent = strings.ReplaceAll(oldContent, editInput.OldString, editInput.NewString)
		if oldContent == newContent && editInput.OldString != "" {
			return "", fmt.Errorf("old_string not found in file")
		}
	} else {
		// Count occurrences first
		count := strings.Count(oldContent, editInput.OldString)
		if count == 0 && editInput.OldString != "" {
			return "", fmt.Errorf("old_string not found in file")
		}
		if count > 1 {
			return "", fmt.Errorf("old_string appears %d times in file. Use replace_all: true to replace all occurrences, or make old_string more specific", count)
		}
		// Replace only first occurrence
		newContent = strings.Replace(oldContent, editInput.OldString, editInput.NewString, 1)
	}

	err = os.WriteFile(editInput.FilePath, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	return "File edited successfully", nil
}

// Keep ReadFileDefinition for backward compatibility
var ReadFileDefinition = ReadDefinition

// Keep EditFileDefinition for backward compatibility
var EditFileDefinition = EditDefinition
