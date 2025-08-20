package auth

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"reapo/internal/config"
	"reapo/internal/logger"
)

// requestLogger middleware to log request headers and body
func requestLogger() option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		// Log request headers
		logger.Debug("Request URL: %s %s", req.Method, req.URL.String())
		logger.Debug("Request Headers:")
		for key, values := range req.Header {
			for _, value := range values {
				logger.Debug("  %s: %s", key, value)
			}
		}

		// Read and log request body if present
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				logger.Error("Error reading request body: %v", err)
			} else {
				logger.Debug("Request Body: %s", string(bodyBytes))
				// Restore the body for the actual request
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		// Call the next middleware/handler
		resp, err := next(req)
		
		if err != nil {
			logger.Error("Request error: %v", err)
		} else if resp != nil {
			logger.Debug("Response Status: %s", resp.Status)
		}

		return resp, err
	}
}

// NewClient creates an authenticated Anthropic client
// Priority: Claude Max (OAuth) > Environment variable
func NewClient() (anthropic.Client, error) {
	// Build common options
	commonOptions := []option.RequestOption{
		option.WithHeader("anthropic-beta", "claude-code-20250219,oauth-2025-04-20,interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14"),
		option.WithHeader("anthropic-dangerous-direct-browser-access", "true"),
		option.WithHeader("x-app", "cli"),
		option.WithHeader("User-Agent", "claude-cli/1.0.83 (external, cli)"),
		option.WithHeader("accept-language", "*"),
		option.WithHeader("sec-fetch-mode", "cors"),
		option.WithHeader("wser-user", ""),
		option.WithQuery("beta", "true"),
	}

	// Add logging middleware if enabled
	if config.GetLogRequests() {
		commonOptions = append(commonOptions, option.WithMiddleware(requestLogger()))
	}

	// First, try to get OAuth token for Claude Max
	token, err := GetAccessToken("anthropic")
	if err == nil && token != "" {
		// Use OAuth token with WithAuthToken
		oauthOptions := append([]option.RequestOption{
			option.WithHeaderDel("X-Api-Key"), // Remove API key header to prevent SDK from using env var
			option.WithAuthToken(token),
		}, commonOptions...)
		return anthropic.NewClient(oauthOptions...), nil
	}

	// Fall back to environment variable
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		return anthropic.NewClient(commonOptions...), nil // SDK will use env var automatically
	}

	return anthropic.Client{}, fmt.Errorf("no authentication method available. Please run /login or set ANTHROPIC_API_KEY")
}

// GetAuthStatus returns the current authentication status
func GetAuthStatus() string {
	// Check OAuth
	token, err := GetAccessToken("anthropic")
	if err == nil && token != "" {
		return "Claude Max (OAuth)"
	}

	// Check environment variable
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return "Environment Variable"
	}

	return "Not authenticated"
}
