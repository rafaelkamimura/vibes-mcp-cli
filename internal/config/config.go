package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds application settings.
// Config holds application settings.
type Config struct {
	// APIKey for the selected provider
	APIKey string
	// BaseURL for the selected provider's API
	BaseURL string
	// Provider selects which MCP provider to use (openai, anthropic, etc.)
	Provider string
	// LogLevel sets the logging level (debug, info, warn, error)
	LogLevel string
	// Templates defines a list of prompt templates for the UI dropdown.
	Templates []string
	// AgentURL is the Vibes Agent backend URL for agent chat and auth
	AgentURL string
	// AuthToken is the JWT access token obtained after login
	AuthToken string
}

// LoadConfig reads config from file or environment.
func LoadConfig(cfgFile string) (*Config, error) {
	// Load environment variables from .env in binary dir and current working dir
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		_ = godotenv.Load(filepath.Join(exeDir, ".env"), ".env")
	} else {
		_ = godotenv.Load(".env")
	}
	v := viper.New()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName(".openai-cli")
		// search for config in current directory first, then home directory
		v.AddConfigPath(".")
		v.AddConfigPath(os.ExpandEnv("$HOME"))
	}
	v.SetEnvPrefix("OPENAI_CLI")
	v.AutomaticEnv()

	// defaults for OpenAI provider and agent backend
	v.SetDefault("base_url", "https://api.openai.com")
	v.SetDefault("provider", "openai")
	v.SetDefault("log_level", "info")
	v.SetDefault("agent_url", "http://localhost:8000")
	v.SetDefault("auth_token", "")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// API key: prefer CLI prefix, fallback to generic OPENAI_API_KEY
	ack := v.GetString("api_key")
	if ack == "" {
		ack = os.Getenv("OPENAI_API_KEY")
	}
	if ack == "" {
		return nil, fmt.Errorf("API key must be set via env OPENAI_CLI_API_KEY, OPENAI_API_KEY, or config file")
	}
	// Base URL: prefer CLI prefix, fallback to generic OPENAI_BASE_URL
	burl := v.GetString("base_url")
	if burl := os.Getenv("OPENAI_BASE_URL"); burl != "" {
		burl = burl
	}
	// Agent URL and auth token for Vibes Agent backend
	agentURL := v.GetString("agent_url")
	authToken := v.GetString("auth_token")

	cfg := &Config{
		APIKey:    ack,
		BaseURL:   burl,
		Provider:  v.GetString("provider"),
		LogLevel:  v.GetString("log_level"),
		Templates: v.GetStringSlice("templates"),
		AgentURL:  agentURL,
		AuthToken: authToken,
	}
	return cfg, nil
}
