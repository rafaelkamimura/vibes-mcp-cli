package components

import (
	"github.com/gdamore/tcell/v2"
)

// PromptTheme defines colors and styles for prompt TUI components
type PromptTheme struct {
	// Primary colors
	Primary       tcell.Color
	Secondary     tcell.Color
	Accent        tcell.Color
	Success       tcell.Color
	Warning       tcell.Color
	Error         tcell.Color
	Info          tcell.Color

	// Background colors
	Background    tcell.Color
	Surface       tcell.Color
	
	// Text colors
	TextPrimary   tcell.Color
	TextSecondary tcell.Color
	TextMuted     tcell.Color
	
	// State colors
	Active        tcell.Color
	Inactive      tcell.Color
	Selected      tcell.Color
	Disabled      tcell.Color
	
	// Category colors
	CategoryGeneral   tcell.Color
	CategoryLanguages tcell.Color
	CategoryWorkflows tcell.Color
	CategoryWorkspace tcell.Color
	CategoryCustom    tcell.Color
}

// DefaultPromptTheme returns the default theme for prompt components
func DefaultPromptTheme() *PromptTheme {
	return &PromptTheme{
		// Primary colors
		Primary:       tcell.ColorBlue,
		Secondary:     tcell.ColorCyan,
		Accent:        tcell.ColorMagenta,
		Success:       tcell.ColorGreen,
		Warning:       tcell.ColorYellow,
		Error:         tcell.ColorRed,
		Info:          tcell.ColorLightBlue,

		// Background colors
		Background:    tcell.ColorBlack,
		Surface:       tcell.ColorDarkSlateGray,
		
		// Text colors
		TextPrimary:   tcell.ColorWhite,
		TextSecondary: tcell.ColorLightGray,
		TextMuted:     tcell.ColorGray,
		
		// State colors
		Active:        tcell.ColorGreen,
		Inactive:      tcell.ColorGray,
		Selected:      tcell.ColorBlue,
		Disabled:      tcell.ColorDarkGray,
		
		// Category colors
		CategoryGeneral:   tcell.ColorWhite,
		CategoryLanguages: tcell.ColorYellow,
		CategoryWorkflows: tcell.ColorCyan,
		CategoryWorkspace: tcell.ColorGreen,
		CategoryCustom:    tcell.ColorMagenta,
	}
}

// GetCategoryColor returns the color for a specific category
func (t *PromptTheme) GetCategoryColor(category string) tcell.Color {
	switch category {
	case "general":
		return t.CategoryGeneral
	case "languages":
		return t.CategoryLanguages
	case "workflows":
		return t.CategoryWorkflows
	case "workspace":
		return t.CategoryWorkspace
	case "custom":
		return t.CategoryCustom
	default:
		return t.TextPrimary
	}
}

// GetStatusColor returns the color for a status or state
func (t *PromptTheme) GetStatusColor(status string) tcell.Color {
	switch status {
	case "success", "valid", "completed":
		return t.Success
	case "warning", "pending":
		return t.Warning
	case "error", "invalid", "failed":
		return t.Error
	case "info", "processing":
		return t.Info
	case "active":
		return t.Active
	case "inactive":
		return t.Inactive
	default:
		return t.TextPrimary
	}
}

// ColorToTags converts a color to tview color tags
func (t *PromptTheme) ColorToTags(color tcell.Color) (string, string) {
	colorName := t.colorToString(color)
	return "[" + colorName + "]", "[white]"
}

// colorToString converts tcell.Color to string for tview tags
func (t *PromptTheme) colorToString(color tcell.Color) string {
	switch color {
	case tcell.ColorRed:
		return "red"
	case tcell.ColorGreen:
		return "green"
	case tcell.ColorYellow:
		return "yellow"
	case tcell.ColorBlue:
		return "blue"
	case tcell.ColorMagenta:
		return "magenta"
	case tcell.ColorCyan:
		return "cyan"
	case tcell.ColorWhite:
		return "white"
	case tcell.ColorLightGray:
		return "lightgray"
	case tcell.ColorGray:
		return "gray"
	case tcell.ColorDarkGray:
		return "darkgray"
	case tcell.ColorLightBlue:
		return "lightblue"
	default:
		return "white"
	}
}

// Icons contains unicode icons for various UI elements
type Icons struct {
	// Template types
	Template     string
	Parameter    string
	Category     string
	
	// Actions
	Generate     string
	Edit         string
	Delete       string
	Copy         string
	Save         string
	
	// Status
	Success      string
	Warning      string
	Error        string
	Info         string
	
	// Navigation
	Folder       string
	File         string
	Back         string
	Forward      string
	Up           string
	Down         string
	
	// Tools
	Search       string
	Filter       string
	Settings     string
	History      string
	Stats        string
	
	// Special
	Star         string
	Heart        string
	Lightning    string
	Gear         string
	Clock        string
}

// DefaultIcons returns the default icon set
func DefaultIcons() *Icons {
	return &Icons{
		// Template types
		Template:     "📝",
		Parameter:    "🔧",
		Category:     "📁",
		
		// Actions
		Generate:     "⚡",
		Edit:         "✏️",
		Delete:       "🗑️",
		Copy:         "📋",
		Save:         "💾",
		
		// Status
		Success:      "✅",
		Warning:      "⚠️",
		Error:        "❌",
		Info:         "ℹ️",
		
		// Navigation
		Folder:       "📁",
		File:         "📄",
		Back:         "◀",
		Forward:      "▶",
		Up:           "▲",
		Down:         "▼",
		
		// Tools
		Search:       "🔍",
		Filter:       "🔽",
		Settings:     "⚙️",
		History:      "📚",
		Stats:        "📊",
		
		// Special
		Star:         "⭐",
		Heart:        "❤️",
		Lightning:    "⚡",
		Gear:         "⚙️",
		Clock:        "🕐",
	}
}