package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthManager struct {
	configPath string
	agentURL   string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type AuthConfig struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(agentURL string) *AuthManager {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".vibes-mcp-cli")
	os.MkdirAll(configDir, 0700)
	
	return &AuthManager{
		configPath: filepath.Join(configDir, "auth.json"),
		agentURL:   agentURL,
	}
}

// Login authenticates with the backend and stores the token
func (am *AuthManager) Login(username, password string) error {
	// Request all available scopes
	scopes := []string{"agent:chat", "agent:stream", "vibe:use", "vibe:manage"}
	
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	data.Set("scope", strings.Join(scopes, " "))
	
	req, err := http.NewRequest("POST", am.agentURL+"/auth/login", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s - %s", resp.Status, string(body))
	}
	
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	
	// Parse token to get expiration
	token, _, err := new(jwt.Parser).ParseUnverified(tokenResp.AccessToken, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("parsing token: %w", err)
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}
	
	exp, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("missing expiration in token")
	}
	
	authConfig := AuthConfig{
		Token:     tokenResp.AccessToken,
		Username:  username,
		ExpiresAt: time.Unix(int64(exp), 0),
	}
	
	return am.saveConfig(authConfig)
}

// GetToken returns the stored token if valid
func (am *AuthManager) GetToken() (string, error) {
	config, err := am.loadConfig()
	if err != nil {
		return "", err
	}
	
	if config.Token == "" {
		return "", fmt.Errorf("no token found")
	}
	
	// Check if token is expired
	if time.Now().After(config.ExpiresAt) {
		return "", fmt.Errorf("token expired")
	}
	
	return config.Token, nil
}

// IsLoggedIn checks if there's a valid token
func (am *AuthManager) IsLoggedIn() bool {
	_, err := am.GetToken()
	return err == nil
}

// Logout removes the stored token
func (am *AuthManager) Logout() error {
	return os.Remove(am.configPath)
}

// GetUsername returns the logged-in username
func (am *AuthManager) GetUsername() string {
	config, err := am.loadConfig()
	if err != nil {
		return ""
	}
	return config.Username
}

func (am *AuthManager) saveConfig(config AuthConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	
	if err := os.WriteFile(am.configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	
	return nil
}

func (am *AuthManager) loadConfig() (AuthConfig, error) {
	var config AuthConfig
	
	data, err := os.ReadFile(am.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, fmt.Errorf("not logged in")
		}
		return config, fmt.Errorf("reading config: %w", err)
	}
	
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("parsing config: %w", err)
	}
	
	return config, nil
}

// AddAuthHeader adds the authorization header to a request
func (am *AuthManager) AddAuthHeader(req *http.Request) error {
	token, err := am.GetToken()
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}