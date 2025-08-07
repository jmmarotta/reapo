package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"reapo/internal/schema"
)

// WebFetchDefinition tool definition
var WebFetchDefinition = ToolDefinition{
	Name: "webfetch",
	Description: `
- Fetches content from a specified URL and processes it using an AI model
- Takes a URL and a prompt as input
- Fetches the URL content, converts HTML to markdown
- Processes the content with the prompt using a small, fast model
- Returns the model's response about the content
- Use this tool when you need to retrieve and analyze web content

Usage notes:
  - IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions. All MCP-provided tools start with "mcp__".
  - The URL must be a fully-formed valid URL
  - HTTP URLs will be automatically upgraded to HTTPS
  - The prompt should describe what information you want to extract from the page
  - This tool is read-only and does not modify any files
  - Results may be summarized if the content is very large
  - Includes a self-cleaning 15-minute cache for faster responses when repeatedly accessing the same URL
  - When a URL redirects to a different host, the tool will inform you and provide the redirect URL in a special format. You should then make a new WebFetch request with the redirect URL to fetch the content.`,
	InputSchema: schema.GenerateSchema[WebFetchInput](),
	Function:    WebFetch,
}

type WebFetchInput struct {
	URL    string `json:"url" jsonschema:"format=uri" jsonschema_description:"The URL to fetch content from"`
	Prompt string `json:"prompt" jsonschema_description:"The prompt to run on the fetched content"`
}

type cacheEntry struct {
	content   string
	timestamp time.Time
}

var (
	webCache      = make(map[string]cacheEntry)
	webCacheMutex sync.RWMutex
)

func init() {
	// Start cache cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cleanWebCache()
		}
	}()
}

func cleanWebCache() {
	webCacheMutex.Lock()
	defer webCacheMutex.Unlock()

	now := time.Now()
	for url, entry := range webCache {
		if now.Sub(entry.timestamp) > 15*time.Minute {
			delete(webCache, url)
		}
	}
}

func WebFetch(input json.RawMessage) (string, error) {
	webFetchInput := WebFetchInput{}
	err := json.Unmarshal(input, &webFetchInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if webFetchInput.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	if webFetchInput.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(webFetchInput.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Upgrade HTTP to HTTPS
	if parsedURL.Scheme == "http" {
		parsedURL.Scheme = "https"
		webFetchInput.URL = parsedURL.String()
	}

	// Check cache
	webCacheMutex.RLock()
	if cached, ok := webCache[webFetchInput.URL]; ok {
		if time.Since(cached.timestamp) < 15*time.Minute {
			webCacheMutex.RUnlock()
			return processContent(cached.content, webFetchInput.Prompt), nil
		}
	}
	webCacheMutex.RUnlock()

	// Fetch the URL
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", webFetchInput.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "reapo/1.0 (AI Agent)")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Check if redirecting to a different host
			if len(via) > 0 {
				originalHost := via[0].URL.Host
				newHost := req.URL.Host
				if originalHost != newHost {
					return fmt.Errorf("redirect to different host: %s", req.URL.String())
				}
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "redirect to different host") {
			redirectURL := strings.TrimPrefix(err.Error(), "redirect to different host: ")
			return fmt.Sprintf("The URL redirects to a different host. Please make a new WebFetch request with this URL: %s", redirectURL), nil
		}
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024*10)) // 10MB limit
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)

	// Simple HTML to markdown conversion
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		content = htmlToMarkdown(content)
	}

	// Cache the content
	webCacheMutex.Lock()
	webCache[webFetchInput.URL] = cacheEntry{
		content:   content,
		timestamp: time.Now(),
	}
	webCacheMutex.Unlock()

	// Process with prompt
	return processContent(content, webFetchInput.Prompt), nil
}

func htmlToMarkdown(html string) string {
	// Very simple HTML to markdown conversion
	// In a real implementation, you'd use a proper HTML parser

	// Remove script and style tags
	html = removeTag(html, "script")
	html = removeTag(html, "style")

	// Convert common tags
	replacements := map[string]string{
		"<h1>": "# ", "</h1>": "\n",
		"<h2>": "## ", "</h2>": "\n",
		"<h3>": "### ", "</h3>": "\n",
		"<h4>": "#### ", "</h4>": "\n",
		"<h5>": "##### ", "</h5>": "\n",
		"<h6>": "###### ", "</h6>": "\n",
		"<p>": "\n", "</p>": "\n",
		"<br>": "\n", "<br/>": "\n", "<br />": "\n",
		"<strong>": "**", "</strong>": "**",
		"<b>": "**", "</b>": "**",
		"<em>": "*", "</em>": "*",
		"<i>": "*", "</i>": "*",
		"<code>": "`", "</code>": "`",
		"<pre>": "```\n", "</pre>": "\n```",
		"<li>": "- ", "</li>": "\n",
		"<ul>": "\n", "</ul>": "\n",
		"<ol>": "\n", "</ol>": "\n",
		"&nbsp;": " ",
		"&lt;":   "<",
		"&gt;":   ">",
		"&amp;":  "&",
		"&quot;": "\"",
		"&#39;":  "'",
	}

	for old, new := range replacements {
		html = strings.ReplaceAll(html, old, new)
	}

	// Remove remaining HTML tags
	result := ""
	inTag := false
	for _, char := range html {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			result += string(char)
		}
	}

	// Clean up multiple newlines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(result)
}

func removeTag(html, tag string) string {
	for {
		start := strings.Index(html, "<"+tag)
		if start == -1 {
			break
		}
		end := strings.Index(html[start:], "</"+tag+">")
		if end == -1 {
			break
		}
		end += start + len("</"+tag+">")
		html = html[:start] + html[end:]
	}
	return html
}

func processContent(content, prompt string) string {
	// In a real implementation, this would call an AI model
	// For now, we'll return a formatted response with the content

	// Truncate content if too long
	const maxContentLength = 5000
	if len(content) > maxContentLength {
		content = content[:maxContentLength] + "\n\n[Content truncated...]"
	}

	return fmt.Sprintf("## Web Content Analysis\n\n**URL Content:**\n%s\n\n**Analysis based on prompt \"%s\":**\nThe content has been fetched successfully. Based on the prompt, here is the extracted content above. Please analyze it according to your specific needs.", content, prompt)
}
