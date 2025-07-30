package files

import (
	"path/filepath"
	"strings"
)

// FileType represents the type of file for syntax highlighting and display
type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeText
	FileTypeCode
	FileTypeMarkdown
	FileTypeJSON
	FileTypeYAML
	FileTypeXML
	FileTypeHTML
	FileTypeCSS
	FileTypeJavaScript
	FileTypeTypeScript
	FileTypeGo
	FileTypePython
	FileTypeJava
	FileTypeC
	FileTypeCPP
	FileTypeRust
	FileTypeShell
	FileTypeSQL
	FileTypeDockerfile
	FileTypeMakefile
	FileTypeConfig
	FileTypeLog
	FileTypeBinary
	FileTypeImage
	FileTypeArchive
	FileTypeExecutable
)

// String returns the string representation of the file type
func (ft FileType) String() string {
	switch ft {
	case FileTypeText:
		return "text"
	case FileTypeCode:
		return "code"
	case FileTypeMarkdown:
		return "markdown"
	case FileTypeJSON:
		return "json"
	case FileTypeYAML:
		return "yaml"
	case FileTypeXML:
		return "xml"
	case FileTypeHTML:
		return "html"
	case FileTypeCSS:
		return "css"
	case FileTypeJavaScript:
		return "javascript"
	case FileTypeTypeScript:
		return "typescript"
	case FileTypeGo:
		return "go"
	case FileTypePython:
		return "python"
	case FileTypeJava:
		return "java"
	case FileTypeC:
		return "c"
	case FileTypeCPP:
		return "cpp"
	case FileTypeRust:
		return "rust"
	case FileTypeShell:
		return "shell"
	case FileTypeSQL:
		return "sql"
	case FileTypeDockerfile:
		return "dockerfile"
	case FileTypeMakefile:
		return "makefile"
	case FileTypeConfig:
		return "config"
	case FileTypeLog:
		return "log"
	case FileTypeBinary:
		return "binary"
	case FileTypeImage:
		return "image"
	case FileTypeArchive:
		return "archive"
	case FileTypeExecutable:
		return "executable"
	default:
		return "unknown"
	}
}

// Icon returns a unicode icon for the file type (for TUI display)
func (ft FileType) Icon() string {
	switch ft {
	case FileTypeText:
		return "📄"
	case FileTypeCode:
		return "📝"
	case FileTypeMarkdown:
		return "📖"
	case FileTypeJSON:
		return "📋"
	case FileTypeYAML:
		return "⚙️"
	case FileTypeXML:
		return "📰"
	case FileTypeHTML:
		return "🌐"
	case FileTypeCSS:
		return "🎨"
	case FileTypeJavaScript, FileTypeTypeScript:
		return "🟨"
	case FileTypeGo:
		return "🐹"
	case FileTypePython:
		return "🐍"
	case FileTypeJava:
		return "☕"
	case FileTypeC, FileTypeCPP:
		return "⚡"
	case FileTypeRust:
		return "🦀"
	case FileTypeShell:
		return "🐚"
	case FileTypeSQL:
		return "🗃️"
	case FileTypeDockerfile:
		return "🐳"
	case FileTypeMakefile:
		return "🔨"
	case FileTypeConfig:
		return "⚙️"
	case FileTypeLog:
		return "📜"
	case FileTypeBinary:
		return "⚫"
	case FileTypeImage:
		return "🖼️"
	case FileTypeArchive:
		return "📦"
	case FileTypeExecutable:
		return "⚙️"
	default:
		return "❓"
	}
}

// IsEditable returns true if the file type can be edited as text
func (ft FileType) IsEditable() bool {
	switch ft {
	case FileTypeText, FileTypeCode, FileTypeMarkdown, FileTypeJSON, FileTypeYAML,
		 FileTypeXML, FileTypeHTML, FileTypeCSS, FileTypeJavaScript, FileTypeTypeScript,
		 FileTypeGo, FileTypePython, FileTypeJava, FileTypeC, FileTypeCPP, FileTypeRust,
		 FileTypeShell, FileTypeSQL, FileTypeDockerfile, FileTypeMakefile, FileTypeConfig:
		return true
	default:
		return false
	}
}

// IsReadable returns true if the file type can be displayed as text
func (ft FileType) IsReadable() bool {
	return ft.IsEditable() || ft == FileTypeLog
}

// SyntaxDetector provides methods for detecting file types and syntax
type SyntaxDetector struct {
	// extensionMap maps file extensions to file types
	extensionMap map[string]FileType
	// filenameMap maps specific filenames to file types
	filenameMap map[string]FileType
}

// NewSyntaxDetector creates a new syntax detector with predefined mappings
func NewSyntaxDetector() *SyntaxDetector {
	sd := &SyntaxDetector{
		extensionMap: make(map[string]FileType),
		filenameMap:  make(map[string]FileType),
	}
	
	sd.initExtensionMappings()
	sd.initFilenameMappings()
	
	return sd
}

// initExtensionMappings sets up the extension to file type mappings
func (sd *SyntaxDetector) initExtensionMappings() {
	// Text files
	sd.extensionMap[".txt"] = FileTypeText
	sd.extensionMap[".md"] = FileTypeMarkdown
	sd.extensionMap[".markdown"] = FileTypeMarkdown
	sd.extensionMap[".rst"] = FileTypeMarkdown
	
	// Data formats
	sd.extensionMap[".json"] = FileTypeJSON
	sd.extensionMap[".yaml"] = FileTypeYAML
	sd.extensionMap[".yml"] = FileTypeYAML
	sd.extensionMap[".xml"] = FileTypeXML
	sd.extensionMap[".toml"] = FileTypeConfig
	sd.extensionMap[".ini"] = FileTypeConfig
	sd.extensionMap[".conf"] = FileTypeConfig
	sd.extensionMap[".cfg"] = FileTypeConfig
	sd.extensionMap[".properties"] = FileTypeConfig
	
	// Web technologies
	sd.extensionMap[".html"] = FileTypeHTML
	sd.extensionMap[".htm"] = FileTypeHTML
	sd.extensionMap[".xhtml"] = FileTypeHTML
	sd.extensionMap[".css"] = FileTypeCSS
	sd.extensionMap[".scss"] = FileTypeCSS
	sd.extensionMap[".sass"] = FileTypeCSS
	sd.extensionMap[".less"] = FileTypeCSS
	sd.extensionMap[".js"] = FileTypeJavaScript
	sd.extensionMap[".jsx"] = FileTypeJavaScript
	sd.extensionMap[".mjs"] = FileTypeJavaScript
	sd.extensionMap[".ts"] = FileTypeTypeScript
	sd.extensionMap[".tsx"] = FileTypeTypeScript
	
	// Programming languages
	sd.extensionMap[".go"] = FileTypeGo
	sd.extensionMap[".py"] = FileTypePython
	sd.extensionMap[".pyw"] = FileTypePython
	sd.extensionMap[".pyc"] = FileTypeBinary
	sd.extensionMap[".java"] = FileTypeJava
	sd.extensionMap[".class"] = FileTypeBinary
	sd.extensionMap[".jar"] = FileTypeArchive
	sd.extensionMap[".c"] = FileTypeC
	sd.extensionMap[".h"] = FileTypeC
	sd.extensionMap[".cpp"] = FileTypeCPP
	sd.extensionMap[".cxx"] = FileTypeCPP
	sd.extensionMap[".cc"] = FileTypeCPP
	sd.extensionMap[".hpp"] = FileTypeCPP
	sd.extensionMap[".hxx"] = FileTypeCPP
	sd.extensionMap[".rs"] = FileTypeRust
	sd.extensionMap[".sh"] = FileTypeShell
	sd.extensionMap[".bash"] = FileTypeShell
	sd.extensionMap[".zsh"] = FileTypeShell
	sd.extensionMap[".fish"] = FileTypeShell
	sd.extensionMap[".sql"] = FileTypeSQL
	
	// Archives and binaries
	sd.extensionMap[".zip"] = FileTypeArchive
	sd.extensionMap[".tar"] = FileTypeArchive
	sd.extensionMap[".gz"] = FileTypeArchive
	sd.extensionMap[".bz2"] = FileTypeArchive
	sd.extensionMap[".xz"] = FileTypeArchive
	sd.extensionMap[".7z"] = FileTypeArchive
	sd.extensionMap[".rar"] = FileTypeArchive
	
	// Images
	sd.extensionMap[".png"] = FileTypeImage
	sd.extensionMap[".jpg"] = FileTypeImage
	sd.extensionMap[".jpeg"] = FileTypeImage
	sd.extensionMap[".gif"] = FileTypeImage
	sd.extensionMap[".bmp"] = FileTypeImage
	sd.extensionMap[".svg"] = FileTypeImage
	sd.extensionMap[".webp"] = FileTypeImage
	sd.extensionMap[".ico"] = FileTypeImage
	
	// Executables
	sd.extensionMap[".exe"] = FileTypeExecutable
	sd.extensionMap[".msi"] = FileTypeExecutable
	sd.extensionMap[".deb"] = FileTypeExecutable
	sd.extensionMap[".rpm"] = FileTypeExecutable
	sd.extensionMap[".dmg"] = FileTypeExecutable
	sd.extensionMap[".app"] = FileTypeExecutable
	
	// Logs
	sd.extensionMap[".log"] = FileTypeLog
	sd.extensionMap[".out"] = FileTypeLog
}

// initFilenameMappings sets up specific filename to file type mappings
func (sd *SyntaxDetector) initFilenameMappings() {
	// Docker
	sd.filenameMap["Dockerfile"] = FileTypeDockerfile
	sd.filenameMap["Dockerfile.dev"] = FileTypeDockerfile
	sd.filenameMap["Dockerfile.prod"] = FileTypeDockerfile
	sd.filenameMap["dockerfile"] = FileTypeDockerfile
	sd.filenameMap[".dockerignore"] = FileTypeConfig
	
	// Make
	sd.filenameMap["Makefile"] = FileTypeMakefile
	sd.filenameMap["makefile"] = FileTypeMakefile
	sd.filenameMap["GNUmakefile"] = FileTypeMakefile
	
	// Git
	sd.filenameMap[".gitignore"] = FileTypeConfig
	sd.filenameMap[".gitattributes"] = FileTypeConfig
	sd.filenameMap[".gitmodules"] = FileTypeConfig
	
	// Go
	sd.filenameMap["go.mod"] = FileTypeGo
	sd.filenameMap["go.sum"] = FileTypeGo
	
	// Node.js
	sd.filenameMap["package.json"] = FileTypeJSON
	sd.filenameMap["package-lock.json"] = FileTypeJSON
	sd.filenameMap["yarn.lock"] = FileTypeConfig
	sd.filenameMap[".nvmrc"] = FileTypeConfig
	
	// Python
	sd.filenameMap["requirements.txt"] = FileTypeConfig
	sd.filenameMap["setup.py"] = FileTypePython
	sd.filenameMap["setup.cfg"] = FileTypeConfig
	sd.filenameMap["pyproject.toml"] = FileTypeConfig
	sd.filenameMap["Pipfile"] = FileTypeConfig
	sd.filenameMap["Pipfile.lock"] = FileTypeConfig
	
	// README files
	sd.filenameMap["README"] = FileTypeMarkdown
	sd.filenameMap["README.md"] = FileTypeMarkdown
	sd.filenameMap["README.txt"] = FileTypeText
	sd.filenameMap["CHANGELOG"] = FileTypeMarkdown
	sd.filenameMap["CHANGELOG.md"] = FileTypeMarkdown
	sd.filenameMap["LICENSE"] = FileTypeText
	sd.filenameMap["LICENSE.md"] = FileTypeMarkdown
	
	// Config files
	sd.filenameMap[".env"] = FileTypeConfig
	sd.filenameMap[".env.example"] = FileTypeConfig
	sd.filenameMap[".editorconfig"] = FileTypeConfig
}

// DetectFileType determines the file type based on filename and extension
func (sd *SyntaxDetector) DetectFileType(filename string) FileType {
	// Check specific filename mappings first
	if fileType, exists := sd.filenameMap[filename]; exists {
		return fileType
	}
	
	// Check base filename without directory
	baseName := filepath.Base(filename)
	if fileType, exists := sd.filenameMap[baseName]; exists {
		return fileType
	}
	
	// Check extension mappings
	ext := strings.ToLower(filepath.Ext(filename))
	if fileType, exists := sd.extensionMap[ext]; exists {
		return fileType
	}
	
	// Default fallback
	if ext == "" {
		return FileTypeText // Files without extension are typically text
	}
	
	return FileTypeUnknown
}

// GetLanguage returns the programming language name for syntax highlighting
func (sd *SyntaxDetector) GetLanguage(filename string) string {
	fileType := sd.DetectFileType(filename)
	
	switch fileType {
	case FileTypeJavaScript, FileTypeTypeScript:
		return "javascript"
	case FileTypeGo:
		return "go"
	case FileTypePython:
		return "python"
	case FileTypeJava:
		return "java"
	case FileTypeC:
		return "c"
	case FileTypeCPP:
		return "cpp"
	case FileTypeRust:
		return "rust"
	case FileTypeShell:
		return "bash"
	case FileTypeSQL:
		return "sql"
	case FileTypeHTML:
		return "html"
	case FileTypeCSS:
		return "css"
	case FileTypeJSON:
		return "json"
	case FileTypeYAML:
		return "yaml"
	case FileTypeXML:
		return "xml"
	case FileTypeMarkdown:
		return "markdown"
	case FileTypeDockerfile:
		return "dockerfile"
	case FileTypeMakefile:
		return "makefile"
	default:
		return "text"
	}
}

// IsTextFile returns true if the file should be treated as a text file
func (sd *SyntaxDetector) IsTextFile(filename string) bool {
	fileType := sd.DetectFileType(filename)
	return fileType.IsReadable()
}

// IsBinaryFile returns true if the file should be treated as a binary file
func (sd *SyntaxDetector) IsBinaryFile(filename string) bool {
	fileType := sd.DetectFileType(filename)
	return fileType == FileTypeBinary || fileType == FileTypeArchive || 
		   fileType == FileTypeExecutable || fileType == FileTypeImage
}