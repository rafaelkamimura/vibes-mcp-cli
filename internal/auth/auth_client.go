package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"openai-cli/internal/client"
)

// AuthAwareClient wraps the standard client with authentication capabilities
type AuthAwareClient struct {
	*client.Client
	authManager *Manager
	agentURL    string
	fallbackKey string
}

// NewAuthAwareClient creates a client that can use either API keys or auth tokens
func NewAuthAwareClient(apiKey, baseURL, agentURL string) (*AuthAwareClient, error) {
	// Create auth manager for agent backend authentication
	var authManager *Manager
	var err error
	if agentURL != "" {
		authManager, err = NewManager(agentURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create auth manager: %w", err)
		}
	}

	authClient := &AuthAwareClient{
		authManager: authManager,
		agentURL:    agentURL,
		fallbackKey: apiKey,
	}

	// Create the base client with custom auth function
	baseClient := client.NewClientWithAuth(baseURL, authClient.authHeaderFunc)

	authClient.Client = baseClient

	return authClient, nil
}

// authHeaderFunc is the authentication function used by the base client
func (c *AuthAwareClient) authHeaderFunc(req *http.Request) {
	// If the request is going to the agent backend and we have auth manager
	if c.authManager != nil && c.isAgentRequest(req.URL.String()) {
		// Try to get a valid token
		if token, err := c.authManager.GetValidToken(req.Context()); err == nil {
			// Use bearer token authentication for agent backend
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		} else {
			// No valid token - don't set auth header
			// The request will likely fail with 401, and calling code should handle login
		}
	} else {
		// For other APIs (OpenAI, Anthropic), use the API key
		if c.fallbackKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.fallbackKey))
		}
	}
}

// isAgentRequest checks if the request is going to the agent backend
func (c *AuthAwareClient) isAgentRequest(requestURL string) bool {
	if c.agentURL == "" {
		return false
	}
	
	// Parse both URLs for proper comparison
	agentParsed, err := url.Parse(c.agentURL)
	if err != nil {
		return false
	}
	
	reqParsed, err := url.Parse(requestURL)
	if err != nil {
		return false
	}
	
	// Compare host and scheme
	return agentParsed.Host == reqParsed.Host && agentParsed.Scheme == reqParsed.Scheme
}

// GetAuthManager returns the auth manager for login operations
func (c *AuthAwareClient) GetAuthManager() *Manager {
	return c.authManager
}

// RequiresAuth checks if authentication is needed for agent backend requests
func (c *AuthAwareClient) RequiresAuth(ctx context.Context) bool {
	if c.authManager == nil {
		return false
	}
	return !c.authManager.IsLoggedIn(ctx)
}