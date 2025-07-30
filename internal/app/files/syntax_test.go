package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyntaxDetector(t *testing.T) {
	detector := NewSyntaxDetector()
	
	assert.NotNil(t, detector)
	assert.NotEmpty(t, detector.extensionMap)
	assert.NotEmpty(t, detector.filenameMap)
}

func TestSyntaxDetector_DetectFileType(t *testing.T) {
	detector := NewSyntaxDetector()
	
	tests := []struct {
		filename     string
		expectedType FileType
	}{
		// Text files
		{"README.txt", FileTypeText},
		{"document.txt", FileTypeText},
		
		// Markdown files
		{"README.md", FileTypeMarkdown},
		{"CHANGELOG.md", FileTypeMarkdown},
		{"doc.markdown", FileTypeMarkdown},
		{"README", FileTypeMarkdown}, // Special filename mapping
		
		// Data formats
		{"config.json", FileTypeJSON},
		{"package.json", FileTypeJSON}, // Special filename mapping
		{"data.yaml", FileTypeYAML},
		{"config.yml", FileTypeYAML},
		{"data.xml", FileTypeXML},
		{"settings.toml", FileTypeConfig},
		{"app.ini", FileTypeConfig},
		{"server.conf", FileTypeConfig},
		{"database.cfg", FileTypeConfig},
		{"app.properties", FileTypeConfig},
		
		// Web technologies
		{"index.html", FileTypeHTML},
		{"page.htm", FileTypeHTML},
		{"template.xhtml", FileTypeHTML},
		{"styles.css", FileTypeCSS},
		{"main.scss", FileTypeCSS},
		{"theme.sass", FileTypeCSS},
		{"base.less", FileTypeCSS},
		{"app.js", FileTypeJavaScript},
		{"component.jsx", FileTypeJavaScript},
		{"module.mjs", FileTypeJavaScript},
		{"main.ts", FileTypeTypeScript},
		{"component.tsx", FileTypeTypeScript},
		
		// Programming languages
		{"main.go", FileTypeGo},
		{"go.mod", FileTypeGo}, // Special filename mapping
		{"go.sum", FileTypeGo}, // Special filename mapping
		{"script.py", FileTypePython},
		{"app.pyw", FileTypePython},
		{"compiled.pyc", FileTypeBinary},
		{"Main.java", FileTypeJava},
		{"classes.class", FileTypeBinary},
		{"library.jar", FileTypeArchive},
		{"program.c", FileTypeC},
		{"header.h", FileTypeC},
		{"main.cpp", FileTypeCPP},
		{"impl.cxx", FileTypeCPP},
		{"code.cc", FileTypeCPP},
		{"header.hpp", FileTypeCPP},
		{"header.hxx", FileTypeCPP},
		{"main.rs", FileTypeRust},
		{"script.sh", FileTypeShell},
		{"init.bash", FileTypeShell},
		{"setup.zsh", FileTypeShell},
		{"config.fish", FileTypeShell},
		{"query.sql", FileTypeSQL},
		
		// Docker and Make
		{"Dockerfile", FileTypeDockerfile}, // Special filename mapping
		{"Dockerfile.dev", FileTypeDockerfile},
		{"dockerfile", FileTypeDockerfile},
		{"Makefile", FileTypeMakefile}, // Special filename mapping
		{"makefile", FileTypeMakefile},
		{"GNUmakefile", FileTypeMakefile},
		
		// Git files
		{".gitignore", FileTypeConfig},
		{".gitattributes", FileTypeConfig},
		{".gitmodules", FileTypeConfig},
		
		// Node.js files
		{"package-lock.json", FileTypeJSON},
		{"yarn.lock", FileTypeConfig},
		{".nvmrc", FileTypeConfig},
		
		// Python files
		{"requirements.txt", FileTypeConfig},
		{"setup.py", FileTypePython},
		{"setup.cfg", FileTypeConfig},
		{"pyproject.toml", FileTypeConfig},
		{"Pipfile", FileTypeConfig},
		{"Pipfile.lock", FileTypeConfig},
		
		// Archives and binaries
		{"archive.zip", FileTypeArchive},
		{"backup.tar", FileTypeArchive},
		{"data.gz", FileTypeArchive},
		{"file.bz2", FileTypeArchive},
		{"compressed.xz", FileTypeArchive},
		{"package.7z", FileTypeArchive},
		{"data.rar", FileTypeArchive},
		
		// Images
		{"photo.png", FileTypeImage},
		{"image.jpg", FileTypeImage},
		{"picture.jpeg", FileTypeImage},
		{"animation.gif", FileTypeImage},
		{"bitmap.bmp", FileTypeImage},
		{"vector.svg", FileTypeImage},
		{"modern.webp", FileTypeImage},
		{"icon.ico", FileTypeImage},
		
		// Executables
		{"program.exe", FileTypeExecutable},
		{"installer.msi", FileTypeExecutable},
		{"package.deb", FileTypeExecutable},
		{"package.rpm", FileTypeExecutable},
		{"installer.dmg", FileTypeExecutable},
		{"Application.app", FileTypeExecutable},
		
		// Logs
		{"application.log", FileTypeLog},
		{"output.out", FileTypeLog},
		
		// Environment and config files
		{".env", FileTypeConfig},
		{".env.example", FileTypeConfig},
		{".editorconfig", FileTypeConfig},
		
		// License files
		{"LICENSE", FileTypeText},
		{"LICENSE.md", FileTypeMarkdown},
		
		// Unknown files
		{"file.unknown", FileTypeUnknown},
		{"data.xyz", FileTypeUnknown},
		
		// Files without extension (should default to text)
		{"README_WITHOUT_EXT", FileTypeText},
		{"script_no_ext", FileTypeText},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detector.DetectFileType(tt.filename)
			assert.Equal(t, tt.expectedType, result, "File type detection failed for %s", tt.filename)
		})
	}
}

func TestSyntaxDetector_GetLanguage(t *testing.T) {
	detector := NewSyntaxDetector()
	
	tests := []struct {
		filename         string
		expectedLanguage string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"Main.java", "java"},
		{"program.c", "c"},
		{"main.cpp", "cpp"},
		{"main.rs", "rust"},
		{"script.sh", "bash"},
		{"query.sql", "sql"},
		{"app.js", "javascript"},
		{"main.ts", "javascript"},
		{"index.html", "html"},
		{"styles.css", "css"},
		{"data.json", "json"},
		{"config.yaml", "yaml"},
		{"data.xml", "xml"},
		{"README.md", "markdown"},
		{"Dockerfile", "dockerfile"},
		{"Makefile", "makefile"},
		{"unknown.xyz", "text"},
		{"binary.exe", "text"},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detector.GetLanguage(tt.filename)
			assert.Equal(t, tt.expectedLanguage, result, "Language detection failed for %s", tt.filename)
		})
	}
}

func TestSyntaxDetector_IsTextFile(t *testing.T) {
	detector := NewSyntaxDetector()
	
	tests := []struct {
		filename string
		isText   bool
	}{
		// Text files
		{"README.txt", true},
		{"main.go", true},
		{"script.py", true},
		{"config.json", true},
		{"styles.css", true},
		{"index.html", true},
		{"README.md", true},
		{"Dockerfile", true},
		{"Makefile", true},
		{"app.log", true},
		
		// Binary files
		{"photo.png", false},
		{"archive.zip", false},
		{"program.exe", false},
		{"library.jar", false},
		{"compiled.pyc", false},
		{"classes.class", false},
		
		// Unknown files (default to non-text for safety)
		{"unknown.xyz", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detector.IsTextFile(tt.filename)
			assert.Equal(t, tt.isText, result, "Text file detection failed for %s", tt.filename)
		})
	}
}

func TestSyntaxDetector_IsBinaryFile(t *testing.T) {
	detector := NewSyntaxDetector()
	
	tests := []struct {
		filename string
		isBinary bool
	}{
		// Binary files
		{"photo.png", true},
		{"image.jpg", true},
		{"archive.zip", true},
		{"program.exe", true},
		{"library.jar", true},
		{"compiled.pyc", true},
		{"classes.class", true},
		{"installer.msi", true},
		{"package.deb", true},
		
		// Text files
		{"README.txt", false},
		{"main.go", false},
		{"script.py", false},
		{"config.json", false},
		{"styles.css", false},
		{"index.html", false},
		{"README.md", false},
		{"Dockerfile", false},
		{"Makefile", false},
		{"app.log", false},
		
		// Unknown files (default to non-binary)
		{"unknown.xyz", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detector.IsBinaryFile(tt.filename)
			assert.Equal(t, tt.isBinary, result, "Binary file detection failed for %s", tt.filename)
		})
	}
}

func TestFileType_String(t *testing.T) {
	tests := []struct {
		fileType FileType
		expected string
	}{
		{FileTypeUnknown, "unknown"},
		{FileTypeText, "text"},
		{FileTypeCode, "code"},
		{FileTypeMarkdown, "markdown"},
		{FileTypeJSON, "json"},
		{FileTypeYAML, "yaml"},
		{FileTypeXML, "xml"},
		{FileTypeHTML, "html"},
		{FileTypeCSS, "css"},
		{FileTypeJavaScript, "javascript"},
		{FileTypeTypeScript, "typescript"},
		{FileTypeGo, "go"},
		{FileTypePython, "python"},
		{FileTypeJava, "java"},
		{FileTypeC, "c"},
		{FileTypeCPP, "cpp"},
		{FileTypeRust, "rust"},
		{FileTypeShell, "shell"},
		{FileTypeSQL, "sql"},
		{FileTypeDockerfile, "dockerfile"},
		{FileTypeMakefile, "makefile"},
		{FileTypeConfig, "config"},
		{FileTypeLog, "log"},
		{FileTypeBinary, "binary"},
		{FileTypeImage, "image"},
		{FileTypeArchive, "archive"},
		{FileTypeExecutable, "executable"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.fileType.String())
		})
	}
}

func TestFileType_Icon(t *testing.T) {
	tests := []struct {
		fileType     FileType
		expectedIcon string
	}{
		{FileTypeUnknown, "❓"},
		{FileTypeText, "📄"},
		{FileTypeCode, "📝"},
		{FileTypeMarkdown, "📖"},
		{FileTypeJSON, "📋"},
		{FileTypeYAML, "⚙️"},
		{FileTypeXML, "📰"},
		{FileTypeHTML, "🌐"},
		{FileTypeCSS, "🎨"},
		{FileTypeJavaScript, "🟨"},
		{FileTypeTypeScript, "🟨"},
		{FileTypeGo, "🐹"},
		{FileTypePython, "🐍"},
		{FileTypeJava, "☕"},
		{FileTypeC, "⚡"},
		{FileTypeCPP, "⚡"},
		{FileTypeRust, "🦀"},
		{FileTypeShell, "🐚"},
		{FileTypeSQL, "🗃️"},
		{FileTypeDockerfile, "🐳"},
		{FileTypeMakefile, "🔨"},
		{FileTypeConfig, "⚙️"},
		{FileTypeLog, "📜"},
		{FileTypeBinary, "⚫"},
		{FileTypeImage, "🖼️"},
		{FileTypeArchive, "📦"},
		{FileTypeExecutable, "⚙️"},
	}
	
	for _, tt := range tests {
		t.Run(tt.fileType.String(), func(t *testing.T) {
			assert.Equal(t, tt.expectedIcon, tt.fileType.Icon())
		})
	}
}

func TestFileType_IsEditable(t *testing.T) {
	editableTypes := []FileType{
		FileTypeText, FileTypeCode, FileTypeMarkdown, FileTypeJSON, FileTypeYAML,
		FileTypeXML, FileTypeHTML, FileTypeCSS, FileTypeJavaScript, FileTypeTypeScript,
		FileTypeGo, FileTypePython, FileTypeJava, FileTypeC, FileTypeCPP, FileTypeRust,
		FileTypeShell, FileTypeSQL, FileTypeDockerfile, FileTypeMakefile, FileTypeConfig,
	}
	
	nonEditableTypes := []FileType{
		FileTypeUnknown, FileTypeLog, FileTypeBinary, FileTypeImage, 
		FileTypeArchive, FileTypeExecutable,
	}
	
	for _, fileType := range editableTypes {
		t.Run("editable_"+fileType.String(), func(t *testing.T) {
			assert.True(t, fileType.IsEditable(), "File type should be editable: %s", fileType.String())
		})
	}
	
	for _, fileType := range nonEditableTypes {
		t.Run("non_editable_"+fileType.String(), func(t *testing.T) {
			assert.False(t, fileType.IsEditable(), "File type should not be editable: %s", fileType.String())
		})
	}
}

func TestFileType_IsReadable(t *testing.T) {
	readableTypes := []FileType{
		FileTypeText, FileTypeCode, FileTypeMarkdown, FileTypeJSON, FileTypeYAML,
		FileTypeXML, FileTypeHTML, FileTypeCSS, FileTypeJavaScript, FileTypeTypeScript,
		FileTypeGo, FileTypePython, FileTypeJava, FileTypeC, FileTypeCPP, FileTypeRust,
		FileTypeShell, FileTypeSQL, FileTypeDockerfile, FileTypeMakefile, FileTypeConfig,
		FileTypeLog, // Log files are readable but not editable
	}
	
	nonReadableTypes := []FileType{
		FileTypeUnknown, FileTypeBinary, FileTypeImage, FileTypeArchive, FileTypeExecutable,
	}
	
	for _, fileType := range readableTypes {
		t.Run("readable_"+fileType.String(), func(t *testing.T) {
			assert.True(t, fileType.IsReadable(), "File type should be readable: %s", fileType.String())
		})
	}
	
	for _, fileType := range nonReadableTypes {
		t.Run("non_readable_"+fileType.String(), func(t *testing.T) {
			assert.False(t, fileType.IsReadable(), "File type should not be readable: %s", fileType.String())
		})
	}
}

func TestSyntaxDetector_FilenamePriority(t *testing.T) {
	detector := NewSyntaxDetector()
	
	// Test that specific filename mappings take priority over extension mappings
	tests := []struct {
		filename     string
		expectedType FileType
		description  string
	}{
		{
			filename:     "Dockerfile",
			expectedType: FileTypeDockerfile,
			description:  "Dockerfile should be detected as dockerfile, not text",
		},
		{
			filename:     "Makefile",
			expectedType: FileTypeMakefile,
			description:  "Makefile should be detected as makefile, not text",
		},
		{
			filename:     "go.mod",
			expectedType: FileTypeGo,
			description:  "go.mod should be detected as Go file",
		},
		{
			filename:     "package.json",
			expectedType: FileTypeJSON,
			description:  "package.json should be detected as JSON",
		},
		{
			filename:     "README",
			expectedType: FileTypeMarkdown,
			description:  "README without extension should be detected as markdown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detector.DetectFileType(tt.filename)
			assert.Equal(t, tt.expectedType, result, tt.description)
		})
	}
}

func TestSyntaxDetector_PathHandling(t *testing.T) {
	detector := NewSyntaxDetector()
	
	// Test that the detector works with full paths, not just filenames
	tests := []struct {
		path         string
		expectedType FileType
	}{
		{"/path/to/main.go", FileTypeGo},
		{"/home/user/documents/README.md", FileTypeMarkdown},
		{"./src/components/App.tsx", FileTypeTypeScript},
		{"../config/settings.yaml", FileTypeYAML},
		{"/var/log/application.log", FileTypeLog},
		{"/usr/bin/program", FileTypeText}, // No extension defaults to text
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detector.DetectFileType(tt.path)
			assert.Equal(t, tt.expectedType, result, "Path-based detection failed for %s", tt.path)
		})
	}
}

func TestSyntaxDetector_CaseInsensitivity(t *testing.T) {
	detector := NewSyntaxDetector()
	
	// Test that extension detection is case-insensitive
	tests := []struct {
		filename     string
		expectedType FileType
	}{
		{"FILE.GO", FileTypeGo},
		{"Script.PY", FileTypePython},
		{"Document.TXT", FileTypeText},
		{"Image.PNG", FileTypeImage},
		{"Archive.ZIP", FileTypeArchive},
		{"Stylesheet.CSS", FileTypeCSS},
		{"Webpage.HTML", FileTypeHTML},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detector.DetectFileType(tt.filename)
			assert.Equal(t, tt.expectedType, result, "Case-insensitive detection failed for %s", tt.filename)
		})
	}
}

func TestSyntaxDetector_EmptyAndSpecialCases(t *testing.T) {
	detector := NewSyntaxDetector()
	
	tests := []struct {
		filename     string
		expectedType FileType
	}{
		{"", FileTypeUnknown},           // Empty filename
		{".", FileTypeUnknown},          // Just a dot
		{"..", FileTypeUnknown},         // Parent directory reference
		{"file.", FileTypeUnknown},      // Filename ending with dot but no extension
		{".hiddenfile", FileTypeText},   // Hidden file without extension (should be text)
		{"..hiddenfile", FileTypeText},  // Hidden file starting with double dot
		{"file..txt", FileTypeText},     // Double dot in filename
		{"file.tar.gz", FileTypeArchive}, // Multiple extensions (should use last one)
	}
	
	for _, tt := range tests {
		t.Run("special_case_"+tt.filename, func(t *testing.T) {
			result := detector.DetectFileType(tt.filename)
			assert.Equal(t, tt.expectedType, result, "Special case detection failed for '%s'", tt.filename)
		})
	}
}

// Benchmark tests for syntax detection performance
func BenchmarkSyntaxDetector_DetectFileType(b *testing.B) {
	detector := NewSyntaxDetector()
	filenames := []string{
		"main.go", "script.py", "index.html", "styles.css", "app.js",
		"config.json", "data.yaml", "README.md", "Dockerfile", "Makefile",
		"photo.png", "archive.zip", "program.exe", "unknown.xyz",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := filenames[i%len(filenames)]
		detector.DetectFileType(filename)
	}
}

func BenchmarkSyntaxDetector_GetLanguage(b *testing.B) {
	detector := NewSyntaxDetector()
	filenames := []string{
		"main.go", "script.py", "index.html", "styles.css", "app.js",
		"config.json", "data.yaml", "README.md", "Dockerfile", "Makefile",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := filenames[i%len(filenames)]
		detector.GetLanguage(filename)
	}
}

func BenchmarkSyntaxDetector_IsTextFile(b *testing.B) {
	detector := NewSyntaxDetector()
	filenames := []string{
		"main.go", "script.py", "photo.png", "archive.zip", "program.exe",
		"config.json", "data.yaml", "README.md", "binary.bin", "text.txt",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := filenames[i%len(filenames)]
		detector.IsTextFile(filename)
	}
}