package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"reapo/internal/schema"
)

// GrepDefinition tool definition
var GrepDefinition = ToolDefinition{
	Name: "grep",
	Description: `A powerful search tool built on ripgrep

  Usage:
  - ALWAYS use Grep for search tasks. NEVER invoke ` + "`grep`" + ` or ` + "`rg`" + ` as a Bash command. The Grep tool has been optimized for correct permissions and access.
  - Supports full regex syntax (e.g., "log.*Error", "function\\s+\\w+")
  - Filter files with glob parameter (e.g., "*.js", "**/*.tsx") or type parameter (e.g., "js", "py", "rust")
  - Output modes: "content" shows matching lines, "files_with_matches" shows only file paths (default), "count" shows match counts
  - Use Task tool for open-ended searches requiring multiple rounds
  - Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping (use ` + "`interface\\{\\}`" + ` to find ` + "`interface{}`" + ` in Go code)
  - Multiline matching: By default patterns match within single lines only. For cross-line patterns like ` + "`struct \\{[\\s\\S]*?field`" + `, use ` + "`multiline: true`" + ``,
	InputSchema: schema.GenerateSchema[GrepInput](),
	Function:    Grep,
}

type GrepInput struct {
	Pattern         string `json:"pattern" jsonschema_description:"The regular expression pattern to search for in file contents"`
	Path            string `json:"path,omitempty" jsonschema_description:"File or directory to search in (rg PATH). Defaults to current working directory."`
	Glob            string `json:"glob,omitempty" jsonschema_description:"Glob pattern to filter files (e.g. \"*.js\", \"*.{ts,tsx}\") - maps to rg --glob"`
	Type            string `json:"type,omitempty" jsonschema_description:"File type to search (rg --type). Common types: js, py, rust, go, java, etc. More efficient than include for standard file types."`
	OutputMode      string `json:"output_mode,omitempty" jsonschema_description:"Output mode: \"content\" shows matching lines (supports -A/-B/-C context, -n line numbers, head_limit), \"files_with_matches\" shows file paths (supports head_limit), \"count\" shows match counts (supports head_limit). Defaults to \"files_with_matches\"."`
	AfterLines      int    `json:"-A,omitempty" jsonschema_description:"Number of lines to show after each match (rg -A). Requires output_mode: \"content\", ignored otherwise."`
	BeforeLines     int    `json:"-B,omitempty" jsonschema_description:"Number of lines to show before each match (rg -B). Requires output_mode: \"content\", ignored otherwise."`
	ContextLines    int    `json:"-C,omitempty" jsonschema_description:"Number of lines to show before and after each match (rg -C). Requires output_mode: \"content\", ignored otherwise."`
	LineNumbers     bool   `json:"-n,omitempty" jsonschema_description:"Show line numbers in output (rg -n). Requires output_mode: \"content\", ignored otherwise."`
	CaseInsensitive bool   `json:"-i,omitempty" jsonschema_description:"Case insensitive search (rg -i)"`
	Multiline       bool   `json:"multiline,omitempty" jsonschema_description:"Enable multiline mode where . matches newlines and patterns can span lines (rg -U --multiline-dotall). Default: false."`
	HeadLimit       int    `json:"head_limit,omitempty" jsonschema_description:"Limit output to first N lines/entries, equivalent to \"| head -N\". Works across all output modes: content (limits output lines), files_with_matches (limits file paths), count (limits count entries). When unspecified, shows all results from ripgrep."`
}

type fileMatch struct {
	path    string
	modTime time.Time
	count   int
	lines   []string
}

func Grep(input json.RawMessage) (string, error) {
	grepInput := GrepInput{}
	err := json.Unmarshal(input, &grepInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if grepInput.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// Set defaults
	if grepInput.OutputMode == "" {
		grepInput.OutputMode = "files_with_matches"
	}

	if grepInput.Path == "" {
		grepInput.Path = "."
	}

	// Compile regex with appropriate flags
	regexFlags := ""
	if grepInput.CaseInsensitive {
		regexFlags = "(?i)"
	}
	if grepInput.Multiline {
		regexFlags += "(?s)"
	}

	regex, err := regexp.Compile(regexFlags + grepInput.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Get file type extensions
	var extensions []string
	if grepInput.Type != "" {
		extensions = getFileTypeExtensions(grepInput.Type)
		if len(extensions) == 0 {
			return "", fmt.Errorf("unknown file type: %s", grepInput.Type)
		}
	}

	// Collect matches
	var matches []fileMatch
	err = filepath.Walk(grepInput.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		// Check glob pattern
		if grepInput.Glob != "" {
			matched, _ := filepath.Match(grepInput.Glob, filepath.Base(path))
			if !matched {
				// Also try matching against relative path
				relPath, _ := filepath.Rel(grepInput.Path, path)
				matched, _ = filepath.Match(grepInput.Glob, relPath)
				if !matched {
					return nil
				}
			}
		}

		// Check file type
		if len(extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			found := false
			for _, e := range extensions {
				if ext == e {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		// Read and search file
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		match := fileMatch{
			path:    path,
			modTime: info.ModTime(),
			lines:   []string{},
		}

		if grepInput.Multiline {
			// Read entire file for multiline matching
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if regex.Match(content) {
				match.count = len(regex.FindAll(content, -1))
				if grepInput.OutputMode == "content" {
					// For multiline, just show the matches
					allMatches := regex.FindAllString(string(content), -1)
					for _, m := range allMatches {
						match.lines = append(match.lines, m)
					}
				}
			}
		} else {
			// Line-by-line matching
			scanner := bufio.NewScanner(file)
			lineNum := 0
			var prevLines []string

			for scanner.Scan() {
				lineNum++
				line := scanner.Text()

				// Keep previous lines for context
				if grepInput.BeforeLines > 0 || grepInput.ContextLines > 0 {
					contextSize := grepInput.BeforeLines
					if grepInput.ContextLines > contextSize {
						contextSize = grepInput.ContextLines
					}
					prevLines = append(prevLines, line)
					if len(prevLines) > contextSize {
						prevLines = prevLines[1:]
					}
				}

				if regex.MatchString(line) {
					match.count++

					if grepInput.OutputMode == "content" {
						// Add before context
						startIdx := 0
						if grepInput.BeforeLines > 0 || grepInput.ContextLines > 0 {
							contextSize := grepInput.BeforeLines
							if grepInput.ContextLines > contextSize {
								contextSize = grepInput.ContextLines
							}
							startIdx = len(prevLines) - contextSize - 1
							if startIdx < 0 {
								startIdx = 0
							}
							for i := startIdx; i < len(prevLines)-1; i++ {
								contextLine := prevLines[i]
								if grepInput.LineNumbers {
									contextLine = fmt.Sprintf("%d:%s", lineNum-len(prevLines)+i+1, contextLine)
								}
								match.lines = append(match.lines, contextLine)
							}
						}

						// Add matching line
						matchLine := line
						if grepInput.LineNumbers {
							matchLine = fmt.Sprintf("%d:%s", lineNum, line)
						}
						match.lines = append(match.lines, matchLine)

						// Read after context
						if grepInput.AfterLines > 0 || grepInput.ContextLines > 0 {
							afterCount := grepInput.AfterLines
							if grepInput.ContextLines > afterCount {
								afterCount = grepInput.ContextLines
							}
							for i := 0; i < afterCount && scanner.Scan(); i++ {
								lineNum++
								afterLine := scanner.Text()
								if grepInput.LineNumbers {
									afterLine = fmt.Sprintf("%d:%s", lineNum, afterLine)
								}
								match.lines = append(match.lines, afterLine)
							}
						}
					}
				}
			}
		}

		if match.count > 0 {
			matches = append(matches, match)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching files: %w", err)
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	// Apply head limit if specified
	if grepInput.HeadLimit > 0 && len(matches) > grepInput.HeadLimit {
		matches = matches[:grepInput.HeadLimit]
	}

	// Format output based on mode
	switch grepInput.OutputMode {
	case "files_with_matches":
		paths := make([]string, len(matches))
		for i, m := range matches {
			paths[i] = m.path
		}
		result, _ := json.Marshal(paths)
		return string(result), nil

	case "count":
		counts := make(map[string]int)
		for _, m := range matches {
			counts[m.path] = m.count
		}
		result, _ := json.Marshal(counts)
		return string(result), nil

	case "content":
		var output strings.Builder
		for i, m := range matches {
			if i > 0 {
				output.WriteString("\n\n")
			}
			output.WriteString(m.path)
			output.WriteString("\n")
			for _, line := range m.lines {
				output.WriteString(line)
				output.WriteString("\n")
			}
		}
		return output.String(), nil

	default:
		return "", fmt.Errorf("invalid output_mode: %s", grepInput.OutputMode)
	}
}

func getFileTypeExtensions(fileType string) []string {
	typeMap := map[string][]string{
		"js":     {".js", ".jsx", ".mjs", ".cjs"},
		"ts":     {".ts", ".tsx", ".mts", ".cts"},
		"py":     {".py", ".pyw", ".pyi"},
		"go":     {".go"},
		"rust":   {".rs"},
		"java":   {".java"},
		"c":      {".c", ".h"},
		"cpp":    {".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		"cs":     {".cs"},
		"php":    {".php"},
		"ruby":   {".rb"},
		"swift":  {".swift"},
		"kotlin": {".kt", ".kts"},
		"scala":  {".scala"},
		"r":      {".r", ".R"},
		"matlab": {".m"},
		"julia":  {".jl"},
		"shell":  {".sh", ".bash", ".zsh", ".fish"},
		"yaml":   {".yaml", ".yml"},
		"json":   {".json"},
		"xml":    {".xml"},
		"html":   {".html", ".htm"},
		"css":    {".css", ".scss", ".sass", ".less"},
		"md":     {".md", ".markdown"},
		"tex":    {".tex"},
		"vim":    {".vim"},
		"lua":    {".lua"},
		"perl":   {".pl", ".pm"},
	}

	return typeMap[strings.ToLower(fileType)]
}
