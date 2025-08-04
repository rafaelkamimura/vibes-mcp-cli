package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"openai-cli/internal/auth"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  "Manage authentication with the Vibes Agent backend",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Vibes Agent backend",
	Long:  "Authenticate with the Vibes Agent backend and store credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentURL := rootCmd.Flag("agent-url").Value.String()
		authManager := auth.NewAuthManager(agentURL)
		
		// Check if already logged in
		if authManager.IsLoggedIn() {
			fmt.Printf("Already logged in as %s\n", authManager.GetUsername())
			fmt.Print("Do you want to re-login? (y/N): ")
			
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			
			if answer != "y" && answer != "yes" {
				return nil
			}
		}
		
		// Get username
		fmt.Print("Username: ")
		reader := bufio.NewReader(os.Stdin)
		username, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading username: %w", err)
		}
		username = strings.TrimSpace(username)
		
		// Get password
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		fmt.Println() // New line after password
		password := string(passwordBytes)
		
		// Attempt login
		fmt.Println("Logging in...")
		if err := authManager.Login(username, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		
		fmt.Printf("Successfully logged in as %s\n", username)
		fmt.Println("Token stored in ~/.vibes-mcp-cli/auth.json")
		
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Vibes Agent backend",
	Long:  "Remove stored authentication credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentURL := rootCmd.Flag("agent-url").Value.String()
		authManager := auth.NewAuthManager(agentURL)
		
		if !authManager.IsLoggedIn() {
			fmt.Println("Not currently logged in")
			return nil
		}
		
		username := authManager.GetUsername()
		
		if err := authManager.Logout(); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}
		
		fmt.Printf("Successfully logged out (was %s)\n", username)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  "Display current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentURL := rootCmd.Flag("agent-url").Value.String()
		authManager := auth.NewAuthManager(agentURL)
		
		if !authManager.IsLoggedIn() {
			fmt.Println("Not logged in")
			return nil
		}
		
		fmt.Printf("Logged in as: %s\n", authManager.GetUsername())
		fmt.Println("Token is valid")
		
		return nil
	},
}

func init() {
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(authCmd)
}