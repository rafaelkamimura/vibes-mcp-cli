package providers

import (
	"fmt"
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
