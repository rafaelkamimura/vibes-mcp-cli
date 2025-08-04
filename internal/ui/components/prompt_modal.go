package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/prompt"
)

// ModalType represents different types of modal dialogs
type ModalType int

const (
	ModalTypeConfirm ModalType = iota
	ModalTypeInput
	ModalTypeSelect
	ModalTypeProgress
	ModalTypeError
	ModalTypeInfo
)

// ModalResult represents the result of a modal dialog
type ModalResult struct {
	Action   string
	Value    string
	Index    int
	Canceled bool
}

// ModalCallback is called when a modal is dismissed
type ModalCallback func(result ModalResult)

// PromptModal provides various modal dialogs for prompt-related operations
type PromptModal struct {
	*tview.Modal

	// Configuration
	config   *PromptUIConfig
	logger   *zap.Logger
	callback ModalCallback

	// Modal state
	modalType ModalType
	title     string
	message   string
	options   []string
	result    ModalResult

	// Input components (for input modals)
	inputField *tview.InputField
	dropDown   *tview.DropDown
	form       *tview.Form

	// Progress components
	progressBar  *tview.TextView
	progressText *tview.TextView
}

// NewPromptModal creates a new modal dialog
func NewPromptModal(config *PromptUIConfig, logger *zap.Logger) *PromptModal {
	if config == nil {
		config = DefaultPromptUIConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	pm := &PromptModal{
		Modal:  tview.NewModal(),
		config: config,
		logger: logger,
	}

	pm.setupStyling()
	return pm
}

// setupStyling applies theme styling to the modal
func (pm *PromptModal) setupStyling() {
	pm.SetBackgroundColor(pm.config.Theme.Surface)
	pm.SetTextColor(pm.config.Theme.TextPrimary)
	pm.SetButtonBackgroundColor(pm.config.Theme.Primary)
	pm.SetButtonTextColor(pm.config.Theme.TextPrimary)
}

// ShowConfirmation shows a confirmation dialog
func (pm *PromptModal) ShowConfirmation(title, message string, callback ModalCallback) {
	pm.modalType = ModalTypeConfirm
	pm.title = title
	pm.message = message
	pm.callback = callback

	pm.SetText(fmt.Sprintf("%s\n\n%s", title, message))
	pm.AddButtons([]string{"Yes", "No", "Cancel"})
	pm.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		result := ModalResult{
			Action:   strings.ToLower(buttonLabel),
			Canceled: buttonLabel == "Cancel",
		}
		if callback != nil {
			callback(result)
		}
	})
}

// ShowError shows an error dialog
func (pm *PromptModal) ShowError(title, message string, callback ModalCallback) {
	pm.modalType = ModalTypeError
	pm.title = title
	pm.message = message
	pm.callback = callback

	errorIcon := pm.config.Icons.Error
	pm.SetText(fmt.Sprintf("%s %s\n\n%s", errorIcon, title, message))
	pm.AddButtons([]string{"OK"})
	pm.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		result := ModalResult{
			Action:   "ok",
			Canceled: false,
		}
		if callback != nil {
			callback(result)
		}
	})
}

// ShowInfo shows an information dialog
func (pm *PromptModal) ShowInfo(title, message string, callback ModalCallback) {
	pm.modalType = ModalTypeInfo
	pm.title = title
	pm.message = message
	pm.callback = callback

	infoIcon := pm.config.Icons.Info
	pm.SetText(fmt.Sprintf("%s %s\n\n%s", infoIcon, title, message))
	pm.AddButtons([]string{"OK"})
	pm.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		result := ModalResult{
			Action:   "ok",
			Canceled: false,
		}
		if callback != nil {
			callback(result)
		}
	})
}

// ShowInput shows an input dialog
func (pm *PromptModal) ShowInput(title, message, placeholder, defaultValue string, callback ModalCallback) {
	pm.modalType = ModalTypeInput
	pm.title = title
	pm.message = message
	pm.callback = callback

	// Create form for input
	pm.form = tview.NewForm()
	pm.form.SetBorder(false)

	pm.inputField = tview.NewInputField()
	pm.inputField.SetLabel("Input: ")
	pm.inputField.SetText(defaultValue)
	if placeholder != "" {
		pm.inputField.SetPlaceholder(placeholder)
	}

	pm.form.AddFormItem(pm.inputField)

	// Create custom modal layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Title and message
	titleText := tview.NewTextView()
	titleText.SetText(fmt.Sprintf("%s\n\n%s", title, message))
	titleText.SetTextAlign(tview.AlignCenter)
	titleText.SetDynamicColors(true)
	
	flex.AddItem(titleText, 0, 1, false)
	flex.AddItem(pm.form, 3, 0, true)

	// Buttons
	buttonFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	
	okButton := tview.NewButton("OK")
	okButton.SetSelectedFunc(func() {
		result := ModalResult{
			Action:   "ok",
			Value:    pm.inputField.GetText(),
			Canceled: false,
		}
		if callback != nil {
			callback(result)
		}
	})

	cancelButton := tview.NewButton("Cancel")
	cancelButton.SetSelectedFunc(func() {
		result := ModalResult{
			Action:   "cancel",
			Canceled: true,
		}
		if callback != nil {
			callback(result)
		}
	})

	buttonFlex.AddItem(okButton, 0, 1, false)
	buttonFlex.AddItem(cancelButton, 0, 1, false)
	
	flex.AddItem(buttonFlex, 3, 0, false)

	// This is a simplified version - in a real implementation,
	// you'd need to replace the modal's content with this flex layout
}

// ShowSelect shows a selection dialog
func (pm *PromptModal) ShowSelect(title, message string, options []string, callback ModalCallback) {
	pm.modalType = ModalTypeSelect
	pm.title = title
	pm.message = message
	pm.options = options
	pm.callback = callback

	// Create form for selection
	pm.form = tview.NewForm()
	pm.form.SetBorder(false)

	pm.dropDown = tview.NewDropDown()
	pm.dropDown.SetLabel("Select: ")
	pm.dropDown.SetOptions(options, nil)

	pm.form.AddFormItem(pm.dropDown)

	// Similar to input modal, but with dropdown
	// Implementation would be similar to ShowInput
}

// ShowProgress shows a progress dialog
func (pm *PromptModal) ShowProgress(title, message string) {
	pm.modalType = ModalTypeProgress
	pm.title = title
	pm.message = message

	pm.progressText = tview.NewTextView()
	pm.progressText.SetText(fmt.Sprintf("%s\n\n%s", title, message))
	pm.progressText.SetTextAlign(tview.AlignCenter)
	pm.progressText.SetDynamicColors(true)

	pm.progressBar = tview.NewTextView()
	pm.progressBar.SetTextAlign(tview.AlignCenter)
	pm.UpdateProgress(0, "Starting...")

	// No buttons for progress dialog - it's closed programmatically
}

// UpdateProgress updates the progress dialog
func (pm *PromptModal) UpdateProgress(percent int, status string) {
	if pm.modalType != ModalTypeProgress {
		return
	}

	// Create a simple text-based progress bar
	barWidth := 30
	filled := (percent * barWidth) / 100
	
	var bar strings.Builder
	bar.WriteString("[")
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString("=")
		} else if i == filled && percent < 100 {
			bar.WriteString(">")
		} else {
			bar.WriteString(" ")
		}
	}
	bar.WriteString("]")

	progressText := fmt.Sprintf("%s %d%%\n\n%s", bar.String(), percent, status)
	pm.progressBar.SetText(progressText)
}

// CloseProgress closes the progress dialog
func (pm *PromptModal) CloseProgress() {
	if pm.modalType == ModalTypeProgress && pm.callback != nil {
		result := ModalResult{
			Action:   "complete",
			Canceled: false,
		}
		pm.callback(result)
	}
}

// Template-specific modal dialogs

// ShowTemplateDeleteConfirm shows confirmation for template deletion
func ShowTemplateDeleteConfirm(template prompt.Template, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := "Delete Template"
	message := fmt.Sprintf("Are you sure you want to delete the template '%s'?\n\nThis action cannot be undone.", template.Name)
	
	modal.ShowConfirmation(title, message, callback)
	return modal
}

// ShowTemplateImportDialog shows template import dialog
func ShowTemplateImportDialog(config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := "Import Template"
	message := "Enter the path to the template file to import:"
	placeholder := "/path/to/template.yaml"
	
	modal.ShowInput(title, message, placeholder, "", callback)
	return modal
}

// ShowTemplateExportDialog shows template export dialog
func ShowTemplateExportDialog(template prompt.Template, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := "Export Template"
	message := fmt.Sprintf("Enter the path to export template '%s':", template.Name)
	defaultPath := fmt.Sprintf("%s.yaml", strings.ReplaceAll(template.Name, " ", "_"))
	
	modal.ShowInput(title, message, "", defaultPath, callback)
	return modal
}

// ShowParameterEditDialog shows parameter editing dialog
func ShowParameterEditDialog(param prompt.TemplateParameter, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	// This would create a complex form for editing all parameter fields
	// For now, we'll just show a simple input for the parameter name
	title := "Edit Parameter"
	message := "Edit parameter name:"
	
	modal.ShowInput(title, message, "Parameter name", param.Name, callback)
	return modal
}

// ShowGenerationProgressDialog shows progress for template generation
func ShowGenerationProgressDialog(templateName string, config *PromptUIConfig) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := "Generating Prompt"
	message := fmt.Sprintf("Generating prompt from template '%s'...", templateName)
	
	modal.ShowProgress(title, message)
	return modal
}

// ShowValidationResultsDialog shows validation results
func ShowValidationResultsDialog(template prompt.Template, issues []string, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	var message strings.Builder
	if len(issues) == 0 {
		message.WriteString(fmt.Sprintf("Template '%s' is valid and ready to use.", template.Name))
	} else {
		message.WriteString(fmt.Sprintf("Template '%s' has %d validation issues:\n\n", template.Name, len(issues)))
		for i, issue := range issues {
			if i >= 5 { // Limit to first 5 issues
				message.WriteString("...and more")
				break
			}
			message.WriteString(fmt.Sprintf("• %s\n", issue))
		}
	}
	
	title := "Validation Results"
	if len(issues) == 0 {
		modal.ShowInfo(title, message.String(), callback)
	} else {
		modal.ShowError(title, message.String(), callback)
	}
	
	return modal
}

// ShowContextRefreshDialog shows context refresh progress
func ShowContextRefreshDialog(config *PromptUIConfig) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := "Refreshing Context"
	message := "Detecting workspace context and updating suggestions..."
	
	modal.ShowProgress(title, message)
	return modal
}

// ShowTemplateSelectionDialog shows template selection dialog
func ShowTemplateSelectionDialog(templates []prompt.Template, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	// Convert templates to options
	options := make([]string, len(templates))
	for i, template := range templates {
		options[i] = fmt.Sprintf("%s - %s", template.Name, template.Description)
	}
	
	title := "Select Template"
	message := "Choose a template to use:"
	
	modal.ShowSelect(title, message, options, callback)
	return modal
}

// ShowCategorySelectionDialog shows category selection dialog
func ShowCategorySelectionDialog(config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	categories := []string{"general", "languages", "workflows", "workspace", "custom"}
	
	title := "Select Category"
	message := "Choose a category for the template:"
	
	modal.ShowSelect(title, message, categories, callback)
	return modal
}

// ShowWorkspaceDetectionResults shows workspace detection results
func ShowWorkspaceDetectionResults(context *prompt.WorkspaceContext, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	var message strings.Builder
	message.WriteString("Detected workspace information:\n\n")
	
	if context.Language != "" {
		message.WriteString(fmt.Sprintf("Language: %s\n", context.Language))
	}
	if context.Framework != "" {
		message.WriteString(fmt.Sprintf("Framework: %s\n", context.Framework))
	}
	if context.Repository != "" {
		message.WriteString(fmt.Sprintf("Repository: %s\n", context.Repository))
	}
	if context.GitBranch != "" {
		message.WriteString(fmt.Sprintf("Git Branch: %s\n", context.GitBranch))
	}
	
	message.WriteString(fmt.Sprintf("\nDirectory: %s", context.WorkingDirectory))
	
	title := "Workspace Detection"
	modal.ShowInfo(title, message.String(), callback)
	
	return modal
}

// ShowGenerationResults shows the results of prompt generation
func ShowGenerationResults(result *prompt.GenerationResult, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	var message strings.Builder
	message.WriteString(fmt.Sprintf("Successfully generated prompt using template '%s'\n\n", result.Template.Name))
	message.WriteString(fmt.Sprintf("Word count: %d\n", result.WordCount))
	message.WriteString(fmt.Sprintf("Character count: %d\n", result.CharCount))
	
	if result.ValidationStatus.Valid {
		message.WriteString(fmt.Sprintf("Validation: Passed (Score: %d/100)\n", result.ValidationStatus.Score))
	} else {
		message.WriteString("Validation: Failed\n")
	}
	
	message.WriteString("\nThe generated prompt has been copied to your clipboard.")
	
	title := "Generation Complete"
	modal.ShowInfo(title, message.String(), callback)
	
	return modal
}

// ShowUnsavedChangesDialog shows dialog for unsaved changes
func ShowUnsavedChangesDialog(itemName string, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := "Unsaved Changes"
	message := fmt.Sprintf("You have unsaved changes to '%s'.\n\nWhat would you like to do?", itemName)
	
	modal.SetText(fmt.Sprintf("%s\n\n%s", title, message))
	modal.AddButtons([]string{"Save", "Discard", "Cancel"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		result := ModalResult{
			Action:   strings.ToLower(buttonLabel),
			Index:    buttonIndex,
			Canceled: buttonLabel == "Cancel",
		}
		if callback != nil {
			callback(result)
		}
	})
	
	return modal
}

// ShowBulkOperationConfirm shows confirmation for bulk operations
func ShowBulkOperationConfirm(operation string, count int, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	title := fmt.Sprintf("Confirm %s", strings.Title(operation))
	message := fmt.Sprintf("Are you sure you want to %s %d items?\n\nThis action cannot be undone.", operation, count)
	
	modal.ShowConfirmation(title, message, callback)
	return modal
}

// ShowErrorWithDetails shows an error dialog with detailed information
func ShowErrorWithDetails(title, summary, details string, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	var message strings.Builder
	message.WriteString(summary)
	
	if details != "" {
		message.WriteString("\n\nDetails:\n")
		message.WriteString(details)
	}
	
	modal.ShowError(title, message.String(), callback)
	return modal
}

// ShowKeyboardShortcuts shows available keyboard shortcuts
func ShowKeyboardShortcuts(shortcuts map[string]string, config *PromptUIConfig, callback ModalCallback) *PromptModal {
	modal := NewPromptModal(config, nil)
	
	var message strings.Builder
	message.WriteString("Available keyboard shortcuts:\n\n")
	
	for key, description := range shortcuts {
		message.WriteString(fmt.Sprintf("%s: %s\n", key, description))
	}
	
	title := "Keyboard Shortcuts"
	modal.ShowInfo(title, message.String(), callback)
	
	return modal
}

// Utility functions for common modal operations

// ConfirmAction shows a simple confirmation dialog
func ConfirmAction(action, item string, config *PromptUIConfig, callback func(confirmed bool)) {
	modal := NewPromptModal(config, nil)
	
	title := fmt.Sprintf("Confirm %s", strings.Title(action))
	message := fmt.Sprintf("Are you sure you want to %s '%s'?", action, item)
	
	modal.ShowConfirmation(title, message, func(result ModalResult) {
		callback(result.Action == "yes")
	})
}

// GetUserInput shows an input dialog and returns the result
func GetUserInput(title, message, placeholder string, config *PromptUIConfig, callback func(input string, canceled bool)) {
	modal := NewPromptModal(config, nil)
	
	modal.ShowInput(title, message, placeholder, "", func(result ModalResult) {
		callback(result.Value, result.Canceled)
	})
}

// SelectFromOptions shows a selection dialog
func SelectFromOptions(title, message string, options []string, config *PromptUIConfig, callback func(selected string, index int, canceled bool)) {
	modal := NewPromptModal(config, nil)
	
	modal.ShowSelect(title, message, options, func(result ModalResult) {
		selectedOption := ""
		if result.Index >= 0 && result.Index < len(options) {
			selectedOption = options[result.Index]
		}
		callback(selectedOption, result.Index, result.Canceled)
	})
}

// ShowNotification shows a simple notification
func ShowNotification(message string, config *PromptUIConfig) {
	modal := NewPromptModal(config, nil)
	modal.ShowInfo("Notification", message, nil)
}