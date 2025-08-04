package providers

import (
	"fmt"
	"openai-cli/internal/auth"
	"openai-cli/internal/client"
	"openai-cli/internal/providers/anthropic"
	"openai-cli/internal/service"
	"strings"
)

const (
	// ProviderOpenAI is the default OpenAI provider
	ProviderOpenAI = "openai"
	// ProviderAnthropic is the Anthropic Claude provider
	ProviderAnthropic = "anthropic"
)

// TestClient may be set by tests to inject a fake APIClient for provider "test".
var TestClient service.APIClient

// NewClient returns a service.APIClient for the given provider.
func NewClient(providerName, apiKey, baseURL string) (service.APIClient, error) {
	return NewClientWithAuth(providerName, apiKey, baseURL, "")
}

// NewClientWithAuth returns a service.APIClient for the given provider with auth support.
func NewClientWithAuth(providerName, apiKey, baseURL, agentURL string) (service.APIClient, error) {
	name := strings.ToLower(providerName)
	// test stub provider
	if name == "test" {
		if TestClient == nil {
			return nil, fmt.Errorf("test client not set")
		}
		return TestClient, nil
	}
	if name == "" {
		name = ProviderOpenAI
	}
	// If agentURL is provided, create auth-aware clients
	if agentURL != "" {
		switch name {
		case ProviderOpenAI:
			return auth.NewAuthAwareClient(apiKey, baseURL, agentURL)
		case ProviderAnthropic:
			// For now, anthropic doesn't support auth-aware client
			// TODO: Implement auth-aware anthropic client
			return anthropic.NewClient(apiKey, baseURL), nil
		default:
			return nil, fmt.Errorf("unknown provider %q", providerName)
		}
	}

	// Standard clients without auth
	switch name {
	case ProviderOpenAI:
		return client.NewClient(apiKey, baseURL), nil
	case ProviderAnthropic:
		// baseURL may be empty; anthropic client will apply default
		return anthropic.NewClient(apiKey, baseURL), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}
}
