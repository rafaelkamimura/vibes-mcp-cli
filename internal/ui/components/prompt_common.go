package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"openai-cli/internal/prompt"
)

// PromptUIConfig holds common configuration for prompt UI components
type PromptUIConfig struct {
	Theme            *PromptTheme
	Icons            *Icons
	ShowIcons        bool
	EnableKeyHelp    bool
	AutoRefreshRate  time.Duration
	MaxHistoryItems  int
	PreviewMaxLines  int
	SearchMinLength  int
}

// DefaultPromptUIConfig returns the default configuration
func DefaultPromptUIConfig() *PromptUIConfig {
	return &PromptUIConfig{
		Theme:            DefaultPromptTheme(),
		Icons:            DefaultIcons(),
		ShowIcons:        true,
		EnableKeyHelp:    true,
		AutoRefreshRate:  time.Second * 5,
		MaxHistoryItems:  100,
		PreviewMaxLines:  50,
		SearchMinLength:  2,
	}
}

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
	Key         tcell.Key
	Rune        rune
	Description string
	Action      func()
}

// KeyBindings holds a collection of key bindings
type KeyBindings struct {
	bindings []KeyBinding
}

// NewKeyBindings creates a new key bindings collection
func NewKeyBindings() *KeyBindings {
	return &KeyBindings{
		bindings: make([]KeyBinding, 0),
	}
}

// Add adds a key binding
func (kb *KeyBindings) Add(key tcell.Key, rune rune, description string, action func()) {
	kb.bindings = append(kb.bindings, KeyBinding{
		Key:         key,
		Rune:        rune,
		Description: description,
		Action:      action,
	})
}

// AddRune adds a rune-based key binding
func (kb *KeyBindings) AddRune(r rune, description string, action func()) {
	kb.Add(tcell.KeyRune, r, description, action)
}

// AddKey adds a key-based binding
func (kb *KeyBindings) AddKey(key tcell.Key, description string, action func()) {
	kb.Add(key, 0, description, action)
}

// Handle processes a key event and executes matching action
func (kb *KeyBindings) Handle(event *tcell.EventKey) bool {
	for _, binding := range kb.bindings {
		if event.Key() == binding.Key {
			if binding.Key == tcell.KeyRune && event.Rune() == binding.Rune {
				if binding.Action != nil {
					binding.Action()
				}
				return true
			} else if binding.Key != tcell.KeyRune {
				if binding.Action != nil {
					binding.Action()
				}
				return true
			}
		}
	}
	return false
}

// GetHelpText returns formatted help text for all bindings
func (kb *KeyBindings) GetHelpText(theme *PromptTheme) string {
	if len(kb.bindings) == 0 {
		return ""
	}

	var parts []string
	for _, binding := range kb.bindings {
		var keyStr string
		if binding.Key == tcell.KeyRune {
			keyStr = string(binding.Rune)
		} else {
			keyStr = kb.keyToString(binding.Key)
		}

		colorStart, colorEnd := theme.ColorToTags(theme.Accent)
		parts = append(parts, fmt.Sprintf("%s%s%s=%s", colorStart, keyStr, colorEnd, binding.Description))
	}

	return strings.Join(parts, " ")
}

// keyToString converts a key to its string representation
func (kb *KeyBindings) keyToString(key tcell.Key) string {
	switch key {
	case tcell.KeyEnter:
		return "Enter"
	case tcell.KeyEsc:
		return "Esc"
	case tcell.KeyTab:
		return "Tab"
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return "Backspace"
	case tcell.KeyDelete:
		return "Del"
	case tcell.KeyInsert:
		return "Ins"
	case tcell.KeyUp:
		return "↑"
	case tcell.KeyDown:
		return "↓"
	case tcell.KeyLeft:
		return "←"
	case tcell.KeyRight:
		return "→"
	case tcell.KeyHome:
		return "Home"
	case tcell.KeyEnd:
		return "End"
	case tcell.KeyPgUp:
		return "PgUp"
	case tcell.KeyPgDn:
		return "PgDn"
	case tcell.KeyF1:
		return "F1"
	case tcell.KeyF2:
		return "F2"
	case tcell.KeyF3:
		return "F3"
	case tcell.KeyF4:
		return "F4"
	case tcell.KeyF5:
		return "F5"
	case tcell.KeyF6:
		return "F6"
	case tcell.KeyF7:
		return "F7"
	case tcell.KeyF8:
		return "F8"
	case tcell.KeyF9:
		return "F9"
	case tcell.KeyF10:
		return "F10"
	case tcell.KeyF11:
		return "F11"
	case tcell.KeyF12:
		return "F12"
	case tcell.KeyCtrlA:
		return "Ctrl+A"
	case tcell.KeyCtrlB:
		return "Ctrl+B"
	case tcell.KeyCtrlC:
		return "Ctrl+C"
	case tcell.KeyCtrlD:
		return "Ctrl+D"
	case tcell.KeyCtrlE:
		return "Ctrl+E"
	case tcell.KeyCtrlF:
		return "Ctrl+F"
	case tcell.KeyCtrlG:
		return "Ctrl+G"
	case tcell.KeyCtrlH:
		return "Ctrl+H"
	case tcell.KeyCtrlI:
		return "Ctrl+I"
	case tcell.KeyCtrlJ:
		return "Ctrl+J"
	case tcell.KeyCtrlK:
		return "Ctrl+K"
	case tcell.KeyCtrlL:
		return "Ctrl+L"
	case tcell.KeyCtrlM:
		return "Ctrl+M"
	case tcell.KeyCtrlN:
		return "Ctrl+N"
	case tcell.KeyCtrlO:
		return "Ctrl+O"
	case tcell.KeyCtrlP:
		return "Ctrl+P"
	case tcell.KeyCtrlQ:
		return "Ctrl+Q"
	case tcell.KeyCtrlR:
		return "Ctrl+R"
	case tcell.KeyCtrlS:
		return "Ctrl+S"
	case tcell.KeyCtrlT:
		return "Ctrl+T"
	case tcell.KeyCtrlU:
		return "Ctrl+U"
	case tcell.KeyCtrlV:
		return "Ctrl+V"
	case tcell.KeyCtrlW:
		return "Ctrl+W"
	case tcell.KeyCtrlX:
		return "Ctrl+X"
	case tcell.KeyCtrlY:
		return "Ctrl+Y"
	case tcell.KeyCtrlZ:
		return "Ctrl+Z"
	default:
		return fmt.Sprintf("Key%d", int(key))
	}
}

// StatusMessage represents a temporary status message
type StatusMessage struct {
	Text     string
	Type     string // success, warning, error, info
	Duration time.Duration
	ShowTime time.Time
}

// NewStatusMessage creates a new status message
func NewStatusMessage(text, msgType string, duration time.Duration) *StatusMessage {
	return &StatusMessage{
		Text:     text,
		Type:     msgType,
		Duration: duration,
		ShowTime: time.Now(),
	}
}

// IsExpired checks if the status message has expired
func (sm *StatusMessage) IsExpired() bool {
	return time.Since(sm.ShowTime) > sm.Duration
}

// FormatForDisplay formats the message for display with colors
func (sm *StatusMessage) FormatForDisplay(theme *PromptTheme) string {
	color := theme.GetStatusColor(sm.Type)
	colorStart, colorEnd := theme.ColorToTags(color)
	return fmt.Sprintf("%s%s%s", colorStart, sm.Text, colorEnd)
}

// TemplateListItem represents a template in a list
type TemplateListItem struct {
	Template    prompt.Template
	DisplayName string
	Description string
	Category    string
	IsSelected  bool
	IsFiltered  bool
}

// NewTemplateListItem creates a new template list item
func NewTemplateListItem(template prompt.Template) *TemplateListItem {
	displayName := template.Name
	if template.Language != "" {
		displayName = fmt.Sprintf("%s (%s)", template.Name, template.Language)
	}
	if template.Framework != "" {
		displayName = fmt.Sprintf("%s [%s]", displayName, template.Framework)
	}

	return &TemplateListItem{
		Template:    template,
		DisplayName: displayName,
		Description: template.Description,
		Category:    template.Category,
		IsSelected:  false,
		IsFiltered:  false,
	}
}

// FormatForList formats the item for display in a list
func (tli *TemplateListItem) FormatForList(config *PromptUIConfig) (string, string) {
	var mainText strings.Builder
	var secondaryText strings.Builder

	// Add icon if enabled
	if config.ShowIcons {
		mainText.WriteString(config.Icons.Template + " ")
	}

	// Add display name with category color
	categoryColor := config.Theme.GetCategoryColor(tli.Category)
	colorStart, colorEnd := config.Theme.ColorToTags(categoryColor)
	mainText.WriteString(fmt.Sprintf("%s%s%s", colorStart, tli.DisplayName, colorEnd))

	// Build secondary text with description and metadata
	if tli.Description != "" {
		secondaryText.WriteString(tli.Description)
	}

	// Add category if different from name
	if tli.Category != "" {
		if secondaryText.Len() > 0 {
			secondaryText.WriteString(" • ")
		}
		secondaryText.WriteString(fmt.Sprintf("[%s]", tli.Category))
	}

	// Add parameter count
	paramCount := len(tli.Template.Parameters)
	if paramCount > 0 {
		if secondaryText.Len() > 0 {
			secondaryText.WriteString(" • ")
		}
		secondaryText.WriteString(fmt.Sprintf("%d params", paramCount))
	}

	return mainText.String(), secondaryText.String()
}

// FilterMatch checks if the item matches a filter string
func (tli *TemplateListItem) FilterMatch(filter string) bool {
	if filter == "" {
		return true
	}

	filter = strings.ToLower(filter)
	
	// Check name, description, category, language, framework, and tags
	checks := []string{
		strings.ToLower(tli.Template.Name),
		strings.ToLower(tli.Template.Description),
		strings.ToLower(tli.Template.Category),
		strings.ToLower(tli.Template.Language),
		strings.ToLower(tli.Template.Framework),
	}

	// Add tags
	for _, tag := range tli.Template.Tags {
		checks = append(checks, strings.ToLower(tag))
	}

	for _, check := range checks {
		if strings.Contains(check, filter) {
			return true
		}
	}

	return false
}

// FormatDuration formats a duration for display
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}

	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}

	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatTime formats a time for display
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	if t.After(now.Add(-24 * time.Hour)) {
		return t.Format("15:04:05")
	}

	if t.After(now.Add(-7 * 24 * time.Hour)) {
		return t.Format("Mon 15:04")
	}

	return t.Format("Jan 2 15:04")
}

// FormatBytes formats bytes for display
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// TruncateText truncates text to a maximum length with ellipsis
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	if maxLen <= 3 {
		return text[:maxLen]
	}

	return text[:maxLen-3] + "..."
}

// WrapText wraps text to a specified width
func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		// If adding this word would exceed the width, start a new line
		if currentLine.Len() > 0 && currentLine.Len()+len(word)+1 > width {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}

		// Add word to current line
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	// Add the last line if it has content
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// CreateModal creates a styled modal dialog
func CreateModal(title string, content tview.Primitive, width, height int, theme *PromptTheme) *tview.Modal {
	modal := tview.NewModal()
	modal.SetText(title)
	modal.SetBackgroundColor(theme.Surface)
	modal.SetTextColor(theme.TextPrimary)
	modal.SetButtonBackgroundColor(theme.Primary)
	modal.SetButtonTextColor(theme.TextPrimary)
	
	return modal
}

// CreateStyledList creates a list with consistent styling
func CreateStyledList(title string, theme *PromptTheme, icons *Icons) *tview.List {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(fmt.Sprintf(" %s %s ", icons.Template, title))
	list.SetTitleAlign(tview.AlignLeft)
	list.SetBackgroundColor(theme.Background)
	list.SetMainTextColor(theme.TextPrimary)
	list.SetSecondaryTextColor(theme.TextSecondary)
	list.SetSelectedTextColor(theme.Selected)
	list.SetSelectedBackgroundColor(theme.Surface)
	
	return list
}

// CreateStyledTextView creates a text view with consistent styling
func CreateStyledTextView(title string, theme *PromptTheme, icons *Icons) *tview.TextView {
	textView := tview.NewTextView()
	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf(" %s %s ", icons.File, title))
	textView.SetTitleAlign(tview.AlignLeft)
	textView.SetBackgroundColor(theme.Background)
	textView.SetTextColor(theme.TextPrimary)
	textView.SetDynamicColors(true)
	textView.SetWordWrap(true)
	
	return textView
}

// CreateStyledInputField creates an input field with consistent styling
func CreateStyledInputField(label string, theme *PromptTheme) *tview.InputField {
	inputField := tview.NewInputField()
	inputField.SetLabel(label)
	inputField.SetFieldBackgroundColor(theme.Surface)
	inputField.SetFieldTextColor(theme.TextPrimary)
	inputField.SetLabelColor(theme.TextSecondary)
	
	return inputField
}