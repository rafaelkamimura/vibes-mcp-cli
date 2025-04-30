package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"openai-cli/internal/config"
)

var (
	cfgFile      string
	providerFlag string
	apiKeyFlag   string
	baseURLFlag  string
	logLevel     string
	cfg          *config.Config
	logger       *zap.Logger
	// Server host and port for MCP server
	serverHost string
	serverPort int
	// serverURL proxies CLI requests to an MCP HTTP server
	serverURL string
	// printCurl outputs the curl command instead of executing
	printCurl bool
)

// rootCmd is the base command for openai-cli
var rootCmd = &cobra.Command{
	Use:   "openai-cli",
	Short: "CLI for OpenAI API",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		// load configuration (env, file)
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			return err
		}
		// CLI flags override config
		if providerFlag != "" {
			cfg.Provider = providerFlag
		}
		if apiKeyFlag != "" {
			cfg.APIKey = apiKeyFlag
		}
		if baseURLFlag != "" {
			cfg.BaseURL = baseURLFlag
		}
		if logLevel != "" {
			cfg.LogLevel = logLevel
		}
		// setup logger
		zapCfg := zap.NewDevelopmentConfig()
		level := zapcore.InfoLevel
		if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
			return fmt.Errorf("invalid log level %q: %w", cfg.LogLevel, err)
		}
		zapCfg.Level = zap.NewAtomicLevelAt(level)
		logger, err = zapCfg.Build()
		if err != nil {
			return err
		}
		return nil
	},
}

// RootCmd exposes the root command for integration testing
var RootCmd = rootCmd

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.openai-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&providerFlag, "provider", "", "provider to use (overrides config: openai, anthropic)")
	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "API base URL (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&serverURL, "server-url", "", "MCP server URL (overrides direct provider)")
	rootCmd.PersistentFlags().BoolVar(&printCurl, "print-curl", false, "Print equivalent curl command and exit")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (overrides config: debug, info, warn, error)")
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(embedCmd)
}
