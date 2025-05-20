package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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

// rootCmd is the base command for the CLI
// getProjectName derives the CLI name from the invoked executable's basename.
func getProjectName() string {
	if exePath, err := os.Executable(); err == nil {
		name := filepath.Base(exePath)
		if ext := filepath.Ext(name); ext != "" {
			name = name[:len(name)-len(ext)]
		}
		return name
	}
	name := filepath.Base(os.Args[0])
	if ext := filepath.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}
	return name
}

var rootCmd = &cobra.Command{
	// Use is overridden at init to match the git repository name
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
	// override CLI use name based on repository
	projectName := getProjectName()
	rootCmd.Use = projectName
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/.%s.yaml)", projectName))
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
