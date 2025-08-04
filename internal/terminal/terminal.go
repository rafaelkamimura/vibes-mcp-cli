package terminal

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Environment represents different execution environments
type Environment int

const (
	EnvironmentUnknown Environment = iota
	EnvironmentInteractive
	EnvironmentCI
	EnvironmentDocker
	EnvironmentHeadless
	EnvironmentSSH
)

// String returns the string representation of the environment
func (e Environment) String() string {
	switch e {
	case EnvironmentInteractive:
		return "interactive"
	case EnvironmentCI:
		return "ci"
	case EnvironmentDocker:
		return "docker"
	case EnvironmentHeadless:
		return "headless"
	case EnvironmentSSH:
		return "ssh"
	default:
		return "unknown"
	}
}

// TerminalInfo contains information about the terminal environment
type TerminalInfo struct {
	HasTTY      bool
	IsTerminal  bool
	Environment Environment
	Width       int
	Height      int
	Term        string
	Details     map[string]string
}

// DetectEnvironment detects the current execution environment
func DetectEnvironment() Environment {
	// Check for CI environment variables
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI",
		"JENKINS_URL", "TRAVIS", "CIRCLECI", "BUILDKITE", "TEAMCITY_VERSION",
		"APPVEYOR", "CODEBUILD_BUILD_ID", "AZURE_HTTP_USER_AGENT",
	}
	
	for _, ciVar := range ciVars {
		if os.Getenv(ciVar) != "" {
			return EnvironmentCI
		}
	}
	
	// Check for Docker environment
	if isRunningInDocker() {
		return EnvironmentDocker
	}
	
	// Check for SSH session
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != "" {
		return EnvironmentSSH
	}
	
	// Check if we have a TTY
	if !HasTTY() {
		return EnvironmentHeadless
	}
	
	return EnvironmentInteractive
}

// isRunningInDocker checks if running inside a Docker container
func isRunningInDocker() bool {
	// Check /.dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	
	// Check cgroup for docker
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}
	
	// Check environment variables
	dockerVars := []string{"DOCKER_CONTAINER", "CONTAINER"}
	for _, dockerVar := range dockerVars {
		if os.Getenv(dockerVar) != "" {
			return true
		}
	}
	
	return false
}

// HasTTY checks if the current process has access to a TTY
func HasTTY() bool {
	// Check if stdin is a terminal
	if !isTerminal(os.Stdin) {
		return false
	}
	
	// Try to open /dev/tty (Unix-like systems)
	if runtime.GOOS != "windows" {
		if file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
			file.Close()
			return true
		}
	}
	
	return false
}

// isTerminal checks if a file is a terminal
func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	
	fileInfo, err := file.Stat()
	if err != nil {
		return false
	}
	
	// On Unix-like systems, check if it's a character device
	if runtime.GOOS != "windows" {
		return (fileInfo.Mode() & os.ModeCharDevice) != 0
	}
	
	// On Windows, we need to use syscalls
	return isWindowsTerminal(file)
}

// isWindowsTerminal checks if a file is a terminal on Windows
func isWindowsTerminal(file *os.File) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	
	// This is a simplified check for Windows
	// In a real implementation, you'd use Windows API calls
	return file.Fd() <= 2 // stdin, stdout, stderr
}

// GetTerminalSize returns the terminal dimensions
func GetTerminalSize() (width, height int) {
	// Default size if we can't determine
	width, height = 80, 24
	
	// Try to get size from environment variables
	if w := os.Getenv("COLUMNS"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil && parsed > 0 {
			width = parsed
		}
	}
	
	if h := os.Getenv("LINES"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			height = parsed
		}
	}
	
	// Try to get size using syscalls (Unix-like systems)
	if runtime.GOOS != "windows" {
		if w, h := getUnixTerminalSize(); w > 0 && h > 0 {
			width, height = w, h
		}
	}
	
	return width, height
}

// getUnixTerminalSize gets terminal size on Unix-like systems
func getUnixTerminalSize() (width, height int) {
	// This would use syscalls like TIOCGWINSZ on real Unix systems
	// For now, return defaults
	return 0, 0
}

// GetTerminalInfo returns comprehensive terminal information
func GetTerminalInfo() *TerminalInfo {
	info := &TerminalInfo{
		HasTTY:     HasTTY(),
		IsTerminal: isTerminal(os.Stdin),
		Environment: DetectEnvironment(),
		Term:       os.Getenv("TERM"),
		Details:    make(map[string]string),
	}
	
	info.Width, info.Height = GetTerminalSize()
	
	// Add environment details
	info.Details["TERM"] = info.Term
	info.Details["COLORTERM"] = os.Getenv("COLORTERM")
	info.Details["SHELL"] = os.Getenv("SHELL")
	info.Details["PWD"] = os.Getenv("PWD")
	
	// Add system details
	info.Details["OS"] = runtime.GOOS
	info.Details["ARCH"] = runtime.GOARCH
	
	// Add process details
	info.Details["PID"] = strconv.Itoa(os.Getpid())
	info.Details["PPID"] = strconv.Itoa(os.Getppid())
	
	// Add Docker/container detection
	if isRunningInDocker() {
		info.Details["CONTAINER"] = "docker"
	}
	
	return info
}

// IsHeadless returns true if running in a headless environment
func IsHeadless() bool {
	env := DetectEnvironment()
	return env == EnvironmentHeadless || env == EnvironmentCI || (!HasTTY() && env == EnvironmentDocker)
}

// CanRunTUI checks if TUI can be safely initialized
func CanRunTUI() (bool, error) {
	info := GetTerminalInfo()
	
	// Check basic requirements
	if !info.HasTTY {
		return false, fmt.Errorf("no TTY available (environment: %s)", info.Environment.String())
	}
	
	if !info.IsTerminal {
		return false, fmt.Errorf("stdin is not a terminal")
	}
	
	// Check terminal size
	if info.Width < 80 || info.Height < 24 {
		return false, fmt.Errorf("terminal too small (%dx%d, minimum 80x24)", info.Width, info.Height)
	}
	
	// Check TERM variable
	if info.Term == "" {
		return false, fmt.Errorf("TERM environment variable not set")
	}
	
	// Warn about potentially problematic environments
	switch info.Environment {
	case EnvironmentCI:
		return false, fmt.Errorf("running in CI environment, TUI not recommended")
	case EnvironmentDocker:
		if !info.HasTTY {
			return false, fmt.Errorf("running in Docker without TTY allocated")
		}
	}
	
	return true, nil
}

// ValidateTerminalEnvironment performs comprehensive terminal validation
func ValidateTerminalEnvironment() error {
	info := GetTerminalInfo()
	
	var issues []string
	
	// Critical issues
	if !info.HasTTY {
		issues = append(issues, fmt.Sprintf("No TTY available (environment: %s)", info.Environment.String()))
	}
	
	if !info.IsTerminal {
		issues = append(issues, "stdin is not a terminal")
	}
	
	// Warning issues
	if info.Width < 80 || info.Height < 24 {
		issues = append(issues, fmt.Sprintf("terminal size too small (%dx%d, recommended 80x24+)", info.Width, info.Height))
	}
	
	if info.Term == "" {
		issues = append(issues, "TERM environment variable not set")
	}
	
	// Environment-specific issues
	switch info.Environment {
	case EnvironmentCI:
		issues = append(issues, "running in CI environment - TUI features may not work properly")
	case EnvironmentDocker:
		if !info.HasTTY {
			issues = append(issues, "running in Docker container without TTY allocated (use 'docker run -it')")
		} else {
			issues = append(issues, "running in Docker container - ensure proper TTY allocation")
		}
	case EnvironmentHeadless:
		issues = append(issues, "running in headless environment - TUI features unavailable")
	}
	
	if len(issues) > 0 {
		return fmt.Errorf("terminal validation failed: %s", strings.Join(issues, "; "))
	}
	
	return nil
}

// TryOpenTTY attempts to open the TTY device safely
func TryOpenTTY() (*os.File, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("TTY opening not supported on Windows")
	}
	
	// First check if we can access /dev/tty
	if _, err := os.Stat("/dev/tty"); err != nil {
		return nil, fmt.Errorf("cannot access /dev/tty: %w", err)
	}
	
	// Try to open it
	file, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot open /dev/tty: %w", err)
	}
	
	return file, nil
}

// GetFallbackInput returns appropriate input source when TTY is not available
func GetFallbackInput() *os.File {
	if HasTTY() {
		return os.Stdin
	}
	
	// In headless environments, we might want to use stdin anyway
	// or return nil to indicate no input is available
	return os.Stdin
}

// GetFallbackOutput returns appropriate output destination when TTY is not available
func GetFallbackOutput() *os.File {
	return os.Stdout
}

// SuggestFix provides user-friendly suggestions for fixing terminal issues
func SuggestFix(err error) string {
	if err == nil {
		return ""
	}
	
	errStr := err.Error()
	info := GetTerminalInfo()
	
	var suggestions []string
	
	if strings.Contains(errStr, "no TTY") || strings.Contains(errStr, "/dev/tty") {
		switch info.Environment {
		case EnvironmentDocker:
			suggestions = append(suggestions, "Run Docker with -it flags: docker run -it your-image")
			suggestions = append(suggestions, "Ensure your Docker container has TTY allocated")
		case EnvironmentCI:
			suggestions = append(suggestions, "Use non-interactive mode in CI environments")
			suggestions = append(suggestions, "Consider using the HTTP server mode instead: vibes-mcp-cli serve")
		case EnvironmentSSH:
			suggestions = append(suggestions, "SSH with TTY allocation: ssh -t user@host")
			suggestions = append(suggestions, "Check SSH configuration for TTY allocation")
		case EnvironmentHeadless:
			suggestions = append(suggestions, "Use server mode for headless environments: vibes-mcp-cli serve")
			suggestions = append(suggestions, "Run in an environment with terminal support")
		}
	}
	
	if strings.Contains(errStr, "terminal too small") {
		suggestions = append(suggestions, "Resize your terminal to at least 80x24 characters")
		suggestions = append(suggestions, "Set COLUMNS and LINES environment variables if needed")
	}
	
	if strings.Contains(errStr, "TERM") {
		suggestions = append(suggestions, "Set TERM environment variable: export TERM=xterm-256color")
		suggestions = append(suggestions, "Check your terminal emulator configuration")
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Use HTTP server mode as alternative: vibes-mcp-cli serve")
		suggestions = append(suggestions, "Run in an interactive terminal environment")
	}
	
	return "Suggestions:\n  • " + strings.Join(suggestions, "\n  • ")
}

// CreateTerminalError creates a user-friendly terminal error with suggestions
func CreateTerminalError(originalErr error, context string) error {
	if originalErr == nil {
		return nil
	}
	
	suggestion := SuggestFix(originalErr)
	if suggestion == "" {
		return fmt.Errorf("%s: %w", context, originalErr)
	}
	
	return fmt.Errorf("%s: %w\n\n%s", context, originalErr, suggestion)
}