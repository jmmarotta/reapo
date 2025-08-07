package tools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"reapo/internal/schema"
)

// WebSearchDefinition tool definition
var WebSearchDefinition = ToolDefinition{
	Name: "websearch",
	Description: `
- Allows reapo to search the web and use the results to inform responses
- Provides up-to-date information for current events and recent data
- Returns search result information formatted as search result blocks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call

Usage notes:
  - Domain filtering is supported to include or block specific websites
  - Web search is only available in the US
  - Account for "Today's date" in <env>. For example, if <env> says "Today's date: 2025-07-01", and the user wants the latest docs, do not use 2024 in the search query. Use 2025.`,
	InputSchema: schema.GenerateSchema[WebSearchInput](),
	Function:    WebSearch,
}

type WebSearchInput struct {
	Query          string   `json:"query" jsonschema:"minLength=2" jsonschema_description:"The search query to use"`
	AllowedDomains []string `json:"allowed_domains,omitempty" jsonschema_description:"Only include search results from these domains"`
	BlockedDomains []string `json:"blocked_domains,omitempty" jsonschema_description:"Never include search results from these domains"`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Domain  string `json:"domain"`
}

func WebSearch(input json.RawMessage) (string, error) {
	webSearchInput := WebSearchInput{}
	err := json.Unmarshal(input, &webSearchInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if webSearchInput.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	if len(webSearchInput.Query) < 2 {
		return "", fmt.Errorf("query must be at least 2 characters long")
	}

	// In a real implementation, this would call a search API
	// For now, we'll return a mock response demonstrating the format

	// Mock search results
	mockResults := []SearchResult{
		{
			Title:   "Example Search Result 1",
			URL:     "https://example.com/result1",
			Snippet: "This is an example search result snippet that would contain relevant information about the search query...",
			Domain:  "example.com",
		},
		{
			Title:   "Example Search Result 2",
			URL:     "https://example.org/result2",
			Snippet: "Another example search result with different content that matches the search query...",
			Domain:  "example.org",
		},
		{
			Title:   "Example Search Result 3",
			URL:     "https://docs.example.com/result3",
			Snippet: "Documentation example that provides detailed information about the topic being searched...",
			Domain:  "docs.example.com",
		},
	}

	// Filter results based on allowed/blocked domains
	var filteredResults []SearchResult
	for _, result := range mockResults {
		parsedURL, err := url.Parse(result.URL)
		if err != nil {
			continue
		}

		domain := parsedURL.Host

		// Check blocked domains
		blocked := false
		for _, blockedDomain := range webSearchInput.BlockedDomains {
			if strings.Contains(domain, blockedDomain) {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}

		// Check allowed domains (if specified)
		if len(webSearchInput.AllowedDomains) > 0 {
			allowed := false
			for _, allowedDomain := range webSearchInput.AllowedDomains {
				if strings.Contains(domain, allowedDomain) {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		filteredResults = append(filteredResults, result)
	}

	// Format results as search result blocks
	var output strings.Builder
	output.WriteString(fmt.Sprintf("## Web Search Results for: \"%s\"\n\n", webSearchInput.Query))

	if len(filteredResults) == 0 {
		output.WriteString("No results found matching your criteria.\n")
		return output.String(), nil
	}

	for i, result := range filteredResults {
		output.WriteString(fmt.Sprintf("### %d. %s\n", i+1, result.Title))
		output.WriteString(fmt.Sprintf("**URL:** %s\n", result.URL))
		output.WriteString(fmt.Sprintf("**Domain:** %s\n", result.Domain))
		output.WriteString(fmt.Sprintf("**Snippet:** %s\n\n", result.Snippet))
	}

	output.WriteString("\n*Note: This is a mock implementation. In production, this would perform actual web searches.*")

	return output.String(), nil
}
