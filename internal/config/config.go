package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const defaultConfigName = ".openai-cli"

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
	// Tools defines a list of available tools for the MCP JSON-RPC proxy.
	Tools []string
	// AgentURL is the Vibes Agent backend URL for agent chat and auth
	AgentURL string
	// AuthToken is the JWT access token obtained after login
	AuthToken string

	// Telemetry settings
	TelemetryEnabled      bool
	TelemetryAPIKey       string
	TelemetryBatchSize    int
	TelemetryFlushInterval time.Duration

	// viper instance for saving config (auth token)
	v *viper.Viper
	// path to the config file used or specified, for persisting settings
	configFile string
}

// Save writes the current configuration (including AuthToken) to the config file.
func (c *Config) Save() error {
	c.v.Set("auth_token", c.AuthToken)

	configFile := c.configFile
	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configFile = filepath.Join(homeDir, defaultConfigName+".yaml")
		c.configFile = configFile
	}
	return c.v.WriteConfigAs(configFile)
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
		v.SetConfigName(defaultConfigName)
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
	// Default tools list for MCP mode
	v.SetDefault("tools", []string{"calculator", "search_web", "weather", "translate", "filesystem"})
	
	// Telemetry defaults
	v.SetDefault("telemetry_enabled", false)
	v.SetDefault("telemetry_api_key", "")
	v.SetDefault("telemetry_batch_size", 50)
	v.SetDefault("telemetry_flush_interval", "30s")

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

	// Telemetry configuration
	telemetryFlushInterval, _ := time.ParseDuration(v.GetString("telemetry_flush_interval"))
	if telemetryFlushInterval == 0 {
		telemetryFlushInterval = 30 * time.Second
	}

	// determine config file path for saving (if loaded or provided via flag)
	configFileUsed := v.ConfigFileUsed()
	if configFileUsed == "" && cfgFile != "" {
		configFileUsed = cfgFile
	}

	cfg := &Config{
		APIKey:     ack,
		BaseURL:    burl,
		Provider:   v.GetString("provider"),
		LogLevel:   v.GetString("log_level"),
		Templates:  v.GetStringSlice("templates"),
		Tools:      v.GetStringSlice("tools"),
		AgentURL:   agentURL,
		AuthToken:  authToken,
		
		// Telemetry settings
		TelemetryEnabled:       v.GetBool("telemetry_enabled"),
		TelemetryAPIKey:        v.GetString("telemetry_api_key"),
		TelemetryBatchSize:     v.GetInt("telemetry_batch_size"),
		TelemetryFlushInterval: telemetryFlushInterval,
		
		v:          v,
		configFile: configFileUsed,
	}
	return cfg, nil
}
