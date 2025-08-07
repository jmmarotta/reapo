package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"reapo/internal/schema"
)

//go:embed prompts/glob.txt
var globPrompt string

// GlobDefinition tool definition
var GlobDefinition = ToolDefinition{
	Name:        "glob",
	Description: globPrompt,
	InputSchema: schema.GenerateSchema[GlobInput](),
	Function:    Glob,
}

type GlobInput struct {
	Pattern string `json:"pattern" jsonschema_description:"The glob pattern to match files against"`
	Path    string `json:"path,omitempty" jsonschema_description:"The directory to search in. If not specified, the current working directory will be used. IMPORTANT: Omit this field to use the default directory. DO NOT enter \"undefined\" or \"null\" - simply omit it for the default behavior. Must be a valid directory path if provided."`
}

type fileWithTime struct {
	path    string
	modTime time.Time
}

func Glob(input json.RawMessage) (string, error) {
	globInput := GlobInput{}
	err := json.Unmarshal(input, &globInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if globInput.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// Determine base path
	basePath := "."
	if globInput.Path != "" {
		basePath = globInput.Path
		// Verify the path exists and is a directory
		info, err := os.Stat(basePath)
		if err != nil {
			return "", fmt.Errorf("path does not exist: %s", basePath)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("path is not a directory: %s", basePath)
		}
	}

	// Convert ** patterns to work with filepath.Walk
	pattern := globInput.Pattern
	hasDoubleWildcard := false
	if len(pattern) > 2 && pattern[:3] == "**/" {
		hasDoubleWildcard = true
		pattern = pattern[3:]
	} else if pattern == "**" {
		hasDoubleWildcard = true
		pattern = "*"
	}

	var matches []fileWithTime

	// Walk the directory tree
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories unless the pattern ends with /
		if info.IsDir() && !strings.HasSuffix(globInput.Pattern, "/") {
			return nil
		}

		// Get relative path from base
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil
		}

		// Skip hidden files/directories (starting with .)
		for _, part := range filepath.SplitList(relPath) {
			if len(part) > 0 && part[0] == '.' {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check if path matches pattern
		matched := false
		if hasDoubleWildcard {
			// For ** patterns, check if any part of the path matches
			matched, _ = filepath.Match(pattern, filepath.Base(path))
			if !matched {
				// Also check against the full relative path
				matched, _ = filepath.Match(pattern, relPath)
			}
		} else {
			// For regular patterns, match against relative path
			matched, _ = filepath.Match(globInput.Pattern, relPath)
		}

		if matched {
			matches = append(matches, fileWithTime{
				path:    path,
				modTime: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	// Extract just the paths
	paths := make([]string, len(matches))
	for i, m := range matches {
		paths[i] = m.path
	}

	// Return as JSON array
	result, err := json.Marshal(paths)
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	return string(result), nil
}
