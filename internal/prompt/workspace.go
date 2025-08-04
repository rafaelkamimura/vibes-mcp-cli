package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// WorkspaceDetectorImpl implements the WorkspaceDetector interface
type WorkspaceDetectorImpl struct {
	logger *zap.Logger
}

// NewWorkspaceDetector creates a new workspace detector
func NewWorkspaceDetector(logger *zap.Logger) WorkspaceDetector {
	return &WorkspaceDetectorImpl{
		logger: logger,
	}
}

// DetectContext detects the current workspace context
func (d *WorkspaceDetectorImpl) DetectContext(ctx context.Context) (*WorkspaceContext, error) {
	d.logger.Debug("Detecting workspace context")

	workingDir, err := os.Getwd()
	if err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeWorkspace,
			Message: "failed to get working directory",
			Cause:   err,
		}
	}

	context := &WorkspaceContext{
		WorkingDirectory: workingDir,
		LastModified:     time.Now(),
		Environment:      make(map[string]string),
	}

	// Detect repository name
	context.Repository = d.detectRepositoryName(workingDir)

	// Detect primary language
	context.Language, err = d.DetectLanguage(workingDir)
	if err != nil {
		d.logger.Warn("Failed to detect language", zap.Error(err))
	}

	// Detect framework if language is known
	if context.Language != "" {
		context.Framework, err = d.DetectFramework(workingDir, context.Language)
		if err != nil {
			d.logger.Warn("Failed to detect framework", zap.Error(err))
		}
	}

	// Get available languages
	context.AvailableLanguages = d.detectAvailableLanguages(workingDir)

	// Get recent files
	context.RecentFiles, err = d.GetRecentFiles(workingDir, 10)
	if err != nil {
		d.logger.Warn("Failed to get recent files", zap.Error(err))
	}

	// Get Git status
	context.GitBranch, context.GitStatus, err = d.GetGitStatus(workingDir)
	if err != nil {
		d.logger.Debug("Not a Git repository or Git not available", zap.Error(err))
	}

	// Get dependencies
	if context.Language != "" {
		context.Dependencies, err = d.GetDependencies(workingDir, context.Language)
		if err != nil {
			d.logger.Warn("Failed to get dependencies", zap.Error(err))
		}
	}

	// Get project structure
	context.ProjectStructure, err = d.GetProjectStructure(workingDir)
	if err != nil {
		d.logger.Warn("Failed to get project structure", zap.Error(err))
	}

	// Collect environment variables
	context.Environment = d.collectEnvironmentInfo()

	d.logger.Info("Workspace context detected",
		zap.String("repository", context.Repository),
		zap.String("language", context.Language),
		zap.String("framework", context.Framework),
		zap.String("branch", context.GitBranch))

	return context, nil
}

// DetectLanguage detects the primary programming language in a directory
func (d *WorkspaceDetectorImpl) DetectLanguage(directory string) (string, error) {
	d.logger.Debug("Detecting language", zap.String("directory", directory))

	// Language detection based on files and configuration
	languageIndicators := map[string][]string{
		"go": {
			"go.mod", "go.sum", "*.go", "Gopkg.toml", "Gopkg.lock",
		},
		"javascript": {
			"package.json", "package-lock.json", "yarn.lock", "*.js", "*.jsx",
		},
		"typescript": {
			"tsconfig.json", "*.ts", "*.tsx", "tslint.json",
		},
		"python": {
			"requirements.txt", "setup.py", "pyproject.toml", "Pipfile", "*.py",
		},
		"java": {
			"pom.xml", "build.gradle", "gradlew", "*.java",
		},
		"rust": {
			"Cargo.toml", "Cargo.lock", "*.rs",
		},
		"c": {
			"Makefile", "CMakeLists.txt", "*.c", "*.h",
		},
		"cpp": {
			"CMakeLists.txt", "*.cpp", "*.cxx", "*.hpp", "*.hxx",
		},
		"php": {
			"composer.json", "composer.lock", "*.php",
		},
		"ruby": {
			"Gemfile", "Gemfile.lock", "*.rb", "Rakefile",
		},
		"swift": {
			"Package.swift", "*.swift",
		},
		"kotlin": {
			"*.kt", "*.kts",
		},
		"csharp": {
			"*.csproj", "*.sln", "*.cs",
		},
	}

	// Score languages based on indicators found
	scores := make(map[string]int)

	for language, indicators := range languageIndicators {
		for _, indicator := range indicators {
			if strings.Contains(indicator, "*") {
				// Handle glob patterns
				matches, err := filepath.Glob(filepath.Join(directory, indicator))
				if err == nil && len(matches) > 0 {
					scores[language] += len(matches)
				}
			} else {
				// Handle specific files
				if _, err := os.Stat(filepath.Join(directory, indicator)); err == nil {
					scores[language] += 10 // Specific files get higher weight
				}
			}
		}
	}

	// Find language with highest score
	var maxScore int
	var detectedLanguage string
	for language, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedLanguage = language
		}
	}

	if detectedLanguage == "" {
		return "", &PromptError{
			Type:    ErrorTypeWorkspace,
			Message: "could not detect programming language",
		}
	}

	d.logger.Info("Language detected",
		zap.String("language", detectedLanguage),
		zap.Int("score", maxScore))

	return detectedLanguage, nil
}

// DetectFramework detects the framework being used for a given language
func (d *WorkspaceDetectorImpl) DetectFramework(directory string, language string) (string, error) {
	d.logger.Debug("Detecting framework",
		zap.String("directory", directory),
		zap.String("language", language))

	frameworkIndicators := map[string]map[string][]string{
		"go": {
			"gin":     {"gin", "github.com/gin-gonic/gin"},
			"echo":    {"echo", "github.com/labstack/echo"},
			"fiber":   {"fiber", "github.com/gofiber/fiber"},
			"chi":     {"chi", "github.com/go-chi/chi"},
			"gorilla": {"gorilla", "github.com/gorilla/mux"},
			"cobra":   {"cobra", "github.com/spf13/cobra"},
			"tview":   {"tview", "github.com/rivo/tview"},
		},
		"javascript": {
			"react":    {"react", "@types/react", "react-dom"},
			"vue":      {"vue", "@vue/cli", "vuex"},
			"angular":  {"@angular/core", "@angular/cli"},
			"express":  {"express", "express-generator"},
			"next":     {"next", "nextjs"},
			"nuxt":     {"nuxt", "@nuxtjs/"},
			"svelte":   {"svelte", "@sveltejs/"},
		},
		"typescript": {
			"react":   {"@types/react", "react"},
			"angular": {"@angular/core", "typescript"},
			"next":    {"next", "typescript"},
		},
		"python": {
			"django":    {"Django", "django", "manage.py"},
			"flask":     {"Flask", "flask", "app.py"},
			"fastapi":   {"fastapi", "uvicorn"},
			"tornado":   {"tornado"},
			"pyramid":   {"pyramid"},
			"bottle":    {"bottle"},
			"sanic":     {"sanic"},
		},
		"java": {
			"spring":     {"spring-boot", "springframework", "pom.xml"},
			"hibernate":  {"hibernate"},
			"struts":     {"struts"},
			"play":       {"play"},
		},
		"rust": {
			"actix":      {"actix-web"},
			"rocket":     {"rocket"},
			"warp":       {"warp"},
			"axum":       {"axum"},
			"tide":       {"tide"},
		},
		"php": {
			"laravel":    {"laravel/framework", "artisan"},
			"symfony":    {"symfony/symfony", "symfony/framework-bundle"},
			"codeigniter": {"codeigniter"},
			"yii":        {"yiisoft/yii2"},
			"cake":       {"cakephp/cakephp"},
		},
		"ruby": {
			"rails":      {"rails", "gem 'rails'"},
			"sinatra":    {"sinatra", "gem 'sinatra'"},
			"hanami":     {"hanami"},
		},
	}

	indicators, exists := frameworkIndicators[language]
	if !exists {
		return "", nil // No framework detection for this language
	}

	// Check package files first
	framework := d.detectFrameworkFromPackageFiles(directory, language, indicators)
	if framework != "" {
		return framework, nil
	}

	// Check source files
	framework = d.detectFrameworkFromSourceFiles(directory, indicators)
	if framework != "" {
		return framework, nil
	}

	return "", nil
}

// GetRecentFiles returns recently modified files
func (d *WorkspaceDetectorImpl) GetRecentFiles(directory string, limit int) ([]string, error) {
	d.logger.Debug("Getting recent files",
		zap.String("directory", directory),
		zap.Int("limit", limit))

	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo

	// Walk directory and collect file info
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common build/cache directories
		skipDirs := []string{"node_modules", "vendor", "target", "build", "dist", ".git"}
		for _, skipDir := range skipDirs {
			if strings.Contains(path, skipDir) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() {
			// Only include source files
			if d.isSourceFile(path) {
				files = append(files, fileInfo{
					path:    path,
					modTime: info.ModTime(),
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// Return requested number of files
	var result []string
	for i, file := range files {
		if i >= limit {
			break
		}
		// Convert to relative path
		relPath, err := filepath.Rel(directory, file.path)
		if err == nil {
			result = append(result, relPath)
		} else {
			result = append(result, file.path)
		}
	}

	return result, nil
}

// GetGitStatus returns Git branch and status information
func (d *WorkspaceDetectorImpl) GetGitStatus(directory string) (string, string, error) {
	d.logger.Debug("Getting Git status", zap.String("directory", directory))

	// Check if it's a Git repository
	gitDir := filepath.Join(directory, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "", "", &PromptError{
			Type:    ErrorTypeWorkspace,
			Message: "not a Git repository",
		}
	}

	// Get current branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = directory
	branchOutput, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get Git branch: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Get status
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = directory
	statusOutput, err := cmd.Output()
	if err != nil {
		return branch, "", fmt.Errorf("failed to get Git status: %w", err)
	}

	status := "clean"
	if len(statusOutput) > 0 {
		status = "dirty"
	}

	return branch, status, nil
}

// GetDependencies returns project dependencies for the given language
func (d *WorkspaceDetectorImpl) GetDependencies(directory string, language string) ([]Dependency, error) {
	d.logger.Debug("Getting dependencies",
		zap.String("directory", directory),
		zap.String("language", language))

	switch language {
	case "go":
		return d.getGoDependencies(directory)
	case "javascript", "typescript":
		return d.getNpmDependencies(directory)
	case "python":
		return d.getPythonDependencies(directory)
	case "java":
		return d.getJavaDependencies(directory)
	case "rust":
		return d.getRustDependencies(directory)
	case "ruby":
		return d.getRubyDependencies(directory)
	case "php":
		return d.getPhpDependencies(directory)
	default:
		return nil, &PromptError{
			Type:    ErrorTypeWorkspace,
			Message: fmt.Sprintf("dependency detection not supported for language: %s", language),
		}
	}
}

// GetProjectStructure returns an overview of the project structure
func (d *WorkspaceDetectorImpl) GetProjectStructure(directory string) ([]string, error) {
	d.logger.Debug("Getting project structure", zap.String("directory", directory))

	var structure []string
	maxDepth := 3 // Limit depth to avoid too much output

	err := d.walkDirectoryStructure(directory, "", 0, maxDepth, &structure)
	if err != nil {
		return nil, err
	}

	return structure, nil
}

// Helper methods

func (d *WorkspaceDetectorImpl) detectRepositoryName(directory string) string {
	// Try to get repository name from Git origin
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = directory
	if output, err := cmd.Output(); err == nil {
		url := strings.TrimSpace(string(output))
		
		// Extract repository name from Git URL
		patterns := []string{
			`github\.com[:/](.+)/(.+?)(?:\.git)?$`,
			`gitlab\.com[:/](.+)/(.+?)(?:\.git)?$`,
			`bitbucket\.org[:/](.+)/(.+?)(?:\.git)?$`,
			`/([^/]+)\.git$`,
			`/([^/]+)/?$`,
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(url); len(matches) >= 2 {
				if len(matches) >= 3 {
					return matches[2] // Return project name for github.com/user/project format
				}
				return matches[1]
			}
		}
	}

	// Fall back to directory name
	return filepath.Base(directory)
}

func (d *WorkspaceDetectorImpl) detectAvailableLanguages(directory string) []string {
	languageFiles := map[string][]string{
		"go":         {"*.go"},
		"javascript": {"*.js", "*.jsx"},
		"typescript": {"*.ts", "*.tsx"},
		"python":     {"*.py"},
		"java":       {"*.java"},
		"rust":       {"*.rs"},
		"c":          {"*.c"},
		"cpp":        {"*.cpp", "*.cxx"},
		"php":        {"*.php"},
		"ruby":       {"*.rb"},
		"swift":      {"*.swift"},
		"kotlin":     {"*.kt"},
		"csharp":     {"*.cs"},
	}

	var languages []string

	for language, patterns := range languageFiles {
		found := false
		for _, pattern := range patterns {
			if matches, err := filepath.Glob(filepath.Join(directory, "**", pattern)); err == nil && len(matches) > 0 {
				found = true
				break
			}
		}
		if found {
			languages = append(languages, language)
		}
	}

	return languages
}

func (d *WorkspaceDetectorImpl) detectFrameworkFromPackageFiles(directory, language string, indicators map[string][]string) string {
	switch language {
	case "go":
		return d.detectGoFramework(directory, indicators)
	case "javascript", "typescript":
		return d.detectNodeFramework(directory, indicators)
	case "python":
		return d.detectPythonFramework(directory, indicators)
	}
	return ""
}

func (d *WorkspaceDetectorImpl) detectFrameworkFromSourceFiles(directory string, indicators map[string][]string) string {
	// Walk through source files looking for imports/includes
	var detectedFramework string
	
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		if !d.isSourceFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentStr := string(content)
		for framework, frameworkIndicators := range indicators {
			for _, indicator := range frameworkIndicators {
				if strings.Contains(contentStr, indicator) {
					detectedFramework = framework
					return filepath.SkipDir // Found framework, stop searching
				}
			}
		}

		return nil
	})

	if err != nil {
		d.logger.Debug("Error walking directory for framework detection", zap.Error(err))
	}

	return detectedFramework
}

func (d *WorkspaceDetectorImpl) detectGoFramework(directory string, indicators map[string][]string) string {
	// Check go.mod file
	goModPath := filepath.Join(directory, "go.mod")
	if content, err := os.ReadFile(goModPath); err == nil {
		contentStr := string(content)
		for framework, frameworkIndicators := range indicators {
			for _, indicator := range frameworkIndicators {
				if strings.Contains(contentStr, indicator) {
					return framework
				}
			}
		}
	}
	return ""
}

func (d *WorkspaceDetectorImpl) detectNodeFramework(directory string, indicators map[string][]string) string {
	// Check package.json
	packagePath := filepath.Join(directory, "package.json")
	if content, err := os.ReadFile(packagePath); err == nil {
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		
		if err := json.Unmarshal(content, &pkg); err == nil {
			allDeps := make(map[string]string)
			for k, v := range pkg.Dependencies {
				allDeps[k] = v
			}
			for k, v := range pkg.DevDependencies {
				allDeps[k] = v
			}

			for framework, frameworkIndicators := range indicators {
				for _, indicator := range frameworkIndicators {
					if _, exists := allDeps[indicator]; exists {
						return framework
					}
				}
			}
		}
	}
	return ""
}

func (d *WorkspaceDetectorImpl) detectPythonFramework(directory string, indicators map[string][]string) string {
	// Check requirements.txt, setup.py, pyproject.toml
	files := []string{"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"}
	
	for _, file := range files {
		filePath := filepath.Join(directory, file)
		if content, err := os.ReadFile(filePath); err == nil {
			contentStr := string(content)
			for framework, frameworkIndicators := range indicators {
				for _, indicator := range frameworkIndicators {
					if strings.Contains(contentStr, indicator) {
						return framework
					}
				}
			}
		}
	}
	return ""
}

func (d *WorkspaceDetectorImpl) isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	sourceExts := []string{
		".go", ".js", ".jsx", ".ts", ".tsx", ".py", ".java", ".rs",
		".c", ".cpp", ".cxx", ".h", ".hpp", ".hxx", ".php", ".rb",
		".swift", ".kt", ".cs", ".scala", ".clj", ".pl", ".sh",
		".yaml", ".yml", ".json", ".xml", ".toml", ".md",
	}

	for _, sourceExt := range sourceExts {
		if ext == sourceExt {
			return true
		}
	}
	return false
}

func (d *WorkspaceDetectorImpl) collectEnvironmentInfo() map[string]string {
	env := make(map[string]string)
	
	// Collect relevant environment variables
	relevantVars := []string{
		"GOPATH", "GOROOT", "GO111MODULE",
		"NODE_ENV", "NODE_PATH",
		"PYTHONPATH", "VIRTUAL_ENV",
		"JAVA_HOME", "MAVEN_HOME",
		"CARGO_HOME", "RUSTUP_HOME",
		"PATH",
	}

	for _, varName := range relevantVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}

	return env
}

func (d *WorkspaceDetectorImpl) walkDirectoryStructure(baseDir, currentPath string, currentDepth, maxDepth int, structure *[]string) error {
	if currentDepth > maxDepth {
		return nil
	}

	fullPath := filepath.Join(baseDir, currentPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Skip hidden files and common build directories
		if strings.HasPrefix(entry.Name(), ".") ||
			entry.Name() == "node_modules" ||
			entry.Name() == "vendor" ||
			entry.Name() == "target" ||
			entry.Name() == "build" ||
			entry.Name() == "dist" {
			continue
		}

		entryPath := filepath.Join(currentPath, entry.Name())
		indent := strings.Repeat("  ", currentDepth)
		
		if entry.IsDir() {
			*structure = append(*structure, indent+"📁 "+entry.Name()+"/")
			d.walkDirectoryStructure(baseDir, entryPath, currentDepth+1, maxDepth, structure)
		} else {
			icon := "📄"
			if d.isSourceFile(entry.Name()) {
				icon = "💾"
			}
			*structure = append(*structure, indent+icon+" "+entry.Name())
		}
	}

	return nil
}

// Dependency detection helper methods

func (d *WorkspaceDetectorImpl) getGoDependencies(directory string) ([]Dependency, error) {
	goModPath := filepath.Join(directory, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	var deps []Dependency
	lines := strings.Split(string(content), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "require (" {
			inRequire = true
			continue
		}
		if line == ")" && inRequire {
			inRequire = false
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				version := parts[1]
				depType := "prod"
				if strings.Contains(line, "// indirect") {
					depType = "indirect"
				}
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Type:    depType,
					Manager: "go",
				})
			}
		}
	}

	return deps, nil
}

func (d *WorkspaceDetectorImpl) getNpmDependencies(directory string) ([]Dependency, error) {
	packagePath := filepath.Join(directory, "package.json")
	content, err := os.ReadFile(packagePath)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}

	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, err
	}

	var deps []Dependency

	for name, version := range pkg.Dependencies {
		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Type:    "prod",
			Manager: "npm",
		})
	}

	for name, version := range pkg.DevDependencies {
		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Type:    "dev",
			Manager: "npm",
		})
	}

	for name, version := range pkg.PeerDependencies {
		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Type:    "peer",
			Manager: "npm",
		})
	}

	return deps, nil
}

func (d *WorkspaceDetectorImpl) getPythonDependencies(directory string) ([]Dependency, error) {
	// Try requirements.txt first
	reqPath := filepath.Join(directory, "requirements.txt")
	if content, err := os.ReadFile(reqPath); err == nil {
		return d.parsePythonRequirements(string(content)), nil
	}

	// Try Pipfile
	pipfilePath := filepath.Join(directory, "Pipfile")
	if _, err := os.Stat(pipfilePath); err == nil {
		// Pipfile parsing would require TOML parser
		return []Dependency{}, nil
	}

	return nil, &PromptError{
		Type:    ErrorTypeWorkspace,
		Message: "no Python dependency file found",
	}
}

func (d *WorkspaceDetectorImpl) parsePythonRequirements(content string) []Dependency {
	var deps []Dependency
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse requirement line (package==version, package>=version, etc.)
		re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)([><=!~]+)([^#\s]+)`)
		matches := re.FindStringSubmatch(line)
		
		if len(matches) >= 3 {
			deps = append(deps, Dependency{
				Name:    matches[1],
				Version: matches[3],
				Type:    "prod",
				Manager: "pip",
			})
		} else {
			// Just package name without version
			name := strings.Fields(line)[0]
			if name != "" {
				deps = append(deps, Dependency{
					Name:    name,
					Version: "*",
					Type:    "prod",
					Manager: "pip",
				})
			}
		}
	}

	return deps
}

func (d *WorkspaceDetectorImpl) getJavaDependencies(directory string) ([]Dependency, error) {
	// Try Maven pom.xml
	pomPath := filepath.Join(directory, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		// XML parsing would require proper XML parser
		return []Dependency{}, nil
	}

	// Try Gradle build.gradle
	gradlePath := filepath.Join(directory, "build.gradle")
	if _, err := os.Stat(gradlePath); err == nil {
		// Gradle parsing would be complex
		return []Dependency{}, nil
	}

	return nil, &PromptError{
		Type:    ErrorTypeWorkspace,
		Message: "no Java dependency file found",
	}
}

func (d *WorkspaceDetectorImpl) getRustDependencies(directory string) ([]Dependency, error) {
	cargoPath := filepath.Join(directory, "Cargo.toml")
	content, err := os.ReadFile(cargoPath)
	if err != nil {
		return nil, err
	}

	// Simple TOML parsing for dependencies section
	var deps []Dependency
	lines := strings.Split(string(content), "\n")
	inDependencies := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "[dependencies]" {
			inDependencies = true
			continue
		}
		
		if strings.HasPrefix(line, "[") && line != "[dependencies]" {
			inDependencies = false
			continue
		}
		
		if inDependencies && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Type:    "prod",
					Manager: "cargo",
				})
			}
		}
	}

	return deps, nil
}

func (d *WorkspaceDetectorImpl) getRubyDependencies(directory string) ([]Dependency, error) {
	gemfilePath := filepath.Join(directory, "Gemfile")
	content, err := os.ReadFile(gemfilePath)
	if err != nil {
		return nil, err
	}

	var deps []Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gem ") {
			// Parse gem line: gem 'name', 'version'
			re := regexp.MustCompile(`gem\s+['"]([^'"]+)['"](?:\s*,\s*['"]([^'"]+)['"])?`)
			matches := re.FindStringSubmatch(line)
			
			if len(matches) >= 2 {
				name := matches[1]
				version := "*"
				if len(matches) >= 3 && matches[2] != "" {
					version = matches[2]
				}
				
				depType := "prod"
				if strings.Contains(line, "group:") && strings.Contains(line, "development") {
					depType = "dev"
				}
				
				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Type:    depType,
					Manager: "bundler",
				})
			}
		}
	}

	return deps, nil
}

func (d *WorkspaceDetectorImpl) getPhpDependencies(directory string) ([]Dependency, error) {
	composerPath := filepath.Join(directory, "composer.json")
	content, err := os.ReadFile(composerPath)
	if err != nil {
		return nil, err
	}

	var composer struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}

	if err := json.Unmarshal(content, &composer); err != nil {
		return nil, err
	}

	var deps []Dependency

	for name, version := range composer.Require {
		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Type:    "prod",
			Manager: "composer",
		})
	}

	for name, version := range composer.RequireDev {
		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Type:    "dev",
			Manager: "composer",
		})
	}

	return deps, nil
}