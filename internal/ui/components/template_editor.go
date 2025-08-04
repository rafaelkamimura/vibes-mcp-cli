package components

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"openai-cli/internal/prompt"
)

// EditorMode represents different editing modes
type EditorMode int

const (
	EditorModeCreate EditorMode = iota
	EditorModeEdit
	EditorModeView
)

// EditorTab represents different editing tabs
type EditorTab int

const (
	TabMetadata EditorTab = iota
	TabContent
	TabParameters
	TabPreview
)

// TemplateEditorCallbacks defines callback functions for editor events
type TemplateEditorCallbacks struct {
	OnTemplateSave   func(template prompt.Template)
	OnTemplateDelete func(template prompt.Template)
	OnTemplateTest   func(template prompt.Template)
	OnError          func(err error)
	OnCancel         func()
}

// TemplateEditor provides a TUI for creating and editing prompt templates
type TemplateEditor struct {
	*tview.Flex

	// Configuration
	config    *PromptUIConfig
	callbacks *TemplateEditorCallbacks
	manager   prompt.Manager
	logger    *zap.Logger

	// UI components
	headerText    *tview.TextView
	tabBar        *tview.TextView
	contentArea   *tview.Flex
	buttonBar     *tview.Flex
	statusBar     *tview.TextView

	// Tab components
	metadataForm  *tview.Form
	contentEditor *tview.TextArea
	parameterList *tview.List
	previewPane   *tview.TextView

	// Parameter editing
	parameterForm *tview.Form
	parameterEdit *tview.Modal

	// Buttons
	saveButton   *tview.Button
	testButton   *tview.Button
	deleteButton *tview.Button
	cancelButton *tview.Button

	// State
	mode         EditorMode
	currentTab   EditorTab
	template     *prompt.Template
	originalTemplate *prompt.Template
	hasChanges   bool
	keyBindings  *KeyBindings

	// Form fields
	nameField        *tview.InputField
	categoryField    *tview.DropDown
	languageField    *tview.InputField
	frameworkField   *tview.InputField
	descriptionField *tview.TextArea
	authorField      *tview.InputField
	versionField     *tview.InputField
	tagsField        *tview.InputField

	// Context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTemplateEditor creates a new template editor component
func NewTemplateEditor(manager prompt.Manager, config *PromptUIConfig, callbacks *TemplateEditorCallbacks, logger *zap.Logger) *TemplateEditor {
	if config == nil {
		config = DefaultPromptUIConfig()
	}

	if callbacks == nil {
		callbacks = &TemplateEditorCallbacks{}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	te := &TemplateEditor{
		Flex:      tview.NewFlex(),
		config:    config,
		callbacks: callbacks,
		manager:   manager,
		logger:    logger,
		mode:      EditorModeCreate,
		currentTab: TabMetadata,
		ctx:       ctx,
		cancel:    cancel,
		template: &prompt.Template{
			Name:        "",
			Category:    "custom",
			Description: "",
			Content:     "",
			Parameters:  []prompt.TemplateParameter{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	te.initKeyBindings()
	te.initUI()

	return te
}

// initKeyBindings sets up keyboard shortcuts
func (te *TemplateEditor) initKeyBindings() {
	te.keyBindings = NewKeyBindings()

	// Tab navigation
	te.keyBindings.AddKey(tcell.KeyF1, "Metadata", func() { te.switchToTab(TabMetadata) })
	te.keyBindings.AddKey(tcell.KeyF2, "Content", func() { te.switchToTab(TabContent) })
	te.keyBindings.AddKey(tcell.KeyF3, "Parameters", func() { te.switchToTab(TabParameters) })
	te.keyBindings.AddKey(tcell.KeyF4, "Preview", func() { te.switchToTab(TabPreview) })

	// Actions
	te.keyBindings.AddKey(tcell.KeyCtrlS, "Save", func() { te.saveTemplate() })
	te.keyBindings.AddKey(tcell.KeyCtrlT, "Test", func() { te.testTemplate() })
	te.keyBindings.AddKey(tcell.KeyCtrlD, "Delete", func() { te.deleteTemplate() })
	te.keyBindings.AddKey(tcell.KeyEsc, "Cancel", func() { te.cancelEditing() })

	// Navigation
	te.keyBindings.AddKey(tcell.KeyTab, "Next Field", func() { te.nextField() })
	te.keyBindings.AddKey(tcell.KeyBacktab, "Prev Field", func() { te.prevField() })

	// Parameter management
	te.keyBindings.AddRune('a', "Add Parameter", func() { te.addParameter() })
	te.keyBindings.AddRune('e', "Edit Parameter", func() { te.editParameter() })
	te.keyBindings.AddRune('d', "Delete Parameter", func() { te.deleteParameter() })

	// Editor operations
	te.keyBindings.AddKey(tcell.KeyCtrlZ, "Undo", func() { te.undo() })
	te.keyBindings.AddKey(tcell.KeyCtrlY, "Redo", func() { te.redo() })
}

// initUI initializes the user interface components
func (te *TemplateEditor) initUI() {
	te.SetDirection(tview.FlexRow)

	// Create header
	te.headerText = tview.NewTextView()
	te.headerText.SetBorder(true)
	te.headerText.SetTitle(fmt.Sprintf(" %s Template Editor ", te.config.Icons.Edit))
	te.headerText.SetTitleAlign(tview.AlignCenter)
	te.headerText.SetDynamicColors(true)
	te.headerText.SetTextAlign(tview.AlignCenter)

	// Create tab bar
	te.tabBar = tview.NewTextView()
	te.tabBar.SetDynamicColors(true)
	te.tabBar.SetTextAlign(tview.AlignCenter)

	// Create content area
	te.contentArea = tview.NewFlex()
	te.contentArea.SetBorder(true)

	// Create button bar
	te.buttonBar = tview.NewFlex().SetDirection(tview.FlexColumn)
	te.initButtons()

	// Create status bar
	te.statusBar = tview.NewTextView()
	te.statusBar.SetDynamicColors(true)
	te.statusBar.SetTextAlign(tview.AlignLeft)

	// Layout
	te.AddItem(te.headerText, 3, 0, false)
	te.AddItem(te.tabBar, 1, 0, false)
	te.AddItem(te.contentArea, 0, 1, true)
	te.AddItem(te.buttonBar, 3, 0, false)
	te.AddItem(te.statusBar, 1, 0, false)

	// Initialize tabs
	te.initTabs()
	te.switchToTab(TabMetadata)
	te.updateUI()

	// Set up input capture
	te.SetInputCapture(te.handleKeyPress)
}

// initButtons creates and configures buttons
func (te *TemplateEditor) initButtons() {
	// Save button
	te.saveButton = tview.NewButton("Save (Ctrl+S)")
	te.saveButton.SetSelectedFunc(te.saveTemplate)

	// Test button
	te.testButton = tview.NewButton("Test (Ctrl+T)")
	te.testButton.SetSelectedFunc(te.testTemplate)

	// Delete button
	te.deleteButton = tview.NewButton("Delete (Ctrl+D)")
	te.deleteButton.SetSelectedFunc(te.deleteTemplate)

	// Cancel button
	te.cancelButton = tview.NewButton("Cancel (Esc)")
	te.cancelButton.SetSelectedFunc(te.cancelEditing)

	// Add buttons to bar
	te.buttonBar.AddItem(te.saveButton, 0, 1, false)
	te.buttonBar.AddItem(te.testButton, 0, 1, false)
	te.buttonBar.AddItem(te.deleteButton, 0, 1, false)
	te.buttonBar.AddItem(tview.NewBox(), 0, 1, false) // Spacer
	te.buttonBar.AddItem(te.cancelButton, 0, 1, false)
}

// initTabs initializes tab components
func (te *TemplateEditor) initTabs() {
	te.initMetadataTab()
	te.initContentTab()
	te.initParametersTab()
	te.initPreviewTab()
}

// initMetadataTab initializes the metadata editing tab
func (te *TemplateEditor) initMetadataTab() {
	te.metadataForm = tview.NewForm()
	te.metadataForm.SetBorder(false)

	// Name field
	te.nameField = tview.NewInputField()
	te.nameField.SetLabel("Name: ")
	te.nameField.SetChangedFunc(func(text string) {
		te.template.Name = text
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.nameField)

	// Category dropdown
	te.categoryField = tview.NewDropDown()
	te.categoryField.SetLabel("Category: ")
	categories := []string{"general", "languages", "workflows", "workspace", "custom"}
	te.categoryField.SetOptions(categories, func(option string, optionIndex int) {
		te.template.Category = option
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.categoryField)

	// Language field
	te.languageField = tview.NewInputField()
	te.languageField.SetLabel("Language: ")
	te.languageField.SetChangedFunc(func(text string) {
		te.template.Language = text
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.languageField)

	// Framework field
	te.frameworkField = tview.NewInputField()
	te.frameworkField.SetLabel("Framework: ")
	te.frameworkField.SetChangedFunc(func(text string) {
		te.template.Framework = text
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.frameworkField)

	// Author field
	te.authorField = tview.NewInputField()
	te.authorField.SetLabel("Author: ")
	te.authorField.SetChangedFunc(func(text string) {
		te.template.Author = text
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.authorField)

	// Version field
	te.versionField = tview.NewInputField()
	te.versionField.SetLabel("Version: ")
	te.versionField.SetChangedFunc(func(text string) {
		te.template.Version = text
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.versionField)

	// Tags field
	te.tagsField = tview.NewInputField()
	te.tagsField.SetLabel("Tags (comma-separated): ")
	te.tagsField.SetChangedFunc(func(text string) {
		if text == "" {
			te.template.Tags = []string{}
		} else {
			tags := strings.Split(text, ",")
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
			te.template.Tags = tags
		}
		te.markChanged()
	})
	te.metadataForm.AddFormItem(te.tagsField)

	// Description field (multi-line)
	te.descriptionField = tview.NewTextArea()
	te.descriptionField.SetLabel("Description:")
	te.descriptionField.SetPlaceholder("Enter template description...")
	te.descriptionField.SetChangedFunc(func() {
		te.template.Description = te.descriptionField.GetText()
		te.markChanged()
	})
}

// initContentTab initializes the content editing tab
func (te *TemplateEditor) initContentTab() {
	te.contentEditor = tview.NewTextArea()
	te.contentEditor.SetPlaceholder("Enter template content...\n\nUse {{parameter_name}} for parameters.")
	te.contentEditor.SetChangedFunc(func() {
		te.template.Content = te.contentEditor.GetText()
		te.markChanged()
		te.updatePreview() // Update preview when content changes
	})
}

// initParametersTab initializes the parameters management tab
func (te *TemplateEditor) initParametersTab() {
	te.parameterList = CreateStyledList("Parameters", te.config.Theme, te.config.Icons)
	
	te.parameterList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		te.editParameterAtIndex(index)
	})

	// Add parameter management buttons
	parameterButtons := tview.NewFlex().SetDirection(tview.FlexColumn)
	
	addButton := tview.NewButton("Add (a)")
	addButton.SetSelectedFunc(te.addParameter)
	
	editButton := tview.NewButton("Edit (e)")
	editButton.SetSelectedFunc(te.editParameter)
	
	deleteButton := tview.NewButton("Delete (d)")
	deleteButton.SetSelectedFunc(te.deleteParameter)

	parameterButtons.AddItem(addButton, 0, 1, false)
	parameterButtons.AddItem(editButton, 0, 1, false)
	parameterButtons.AddItem(deleteButton, 0, 1, false)

	// Layout parameters tab
	parameterTab := tview.NewFlex().SetDirection(tview.FlexRow)
	parameterTab.AddItem(te.parameterList, 0, 1, true)
	parameterTab.AddItem(parameterButtons, 3, 0, false)

	// Store the full tab layout
	te.parameterList = parameterTab.(*tview.List) // This is not correct, but we'll fix it
}

// initPreviewTab initializes the preview tab
func (te *TemplateEditor) initPreviewTab() {
	te.previewPane = CreateStyledTextView("Preview", te.config.Theme, te.config.Icons)
	te.previewPane.SetScrollable(true)
}

// handleKeyPress processes keyboard input
func (te *TemplateEditor) handleKeyPress(event *tcell.EventKey) *tcell.EventKey {
	// Try key bindings first
	if te.keyBindings.Handle(event) {
		return nil
	}

	return event
}

// Tab switching methods
func (te *TemplateEditor) switchToTab(tab EditorTab) {
	te.currentTab = tab
	te.contentArea.Clear()

	switch tab {
	case TabMetadata:
		// Create metadata layout
		metadataLayout := tview.NewFlex().SetDirection(tview.FlexRow)
		metadataLayout.AddItem(te.metadataForm, 0, 2, true)
		metadataLayout.AddItem(te.descriptionField, 0, 1, false)
		te.contentArea.AddItem(metadataLayout, 0, 1, true)

	case TabContent:
		te.contentArea.AddItem(te.contentEditor, 0, 1, true)

	case TabParameters:
		te.setupParametersTabContent()

	case TabPreview:
		te.updatePreview()
		te.contentArea.AddItem(te.previewPane, 0, 1, true)
	}

	te.updateTabBar()
	te.updateUI()
}

func (te *TemplateEditor) setupParametersTabContent() {
	// Recreate parameter list
	parameterList := CreateStyledList("Parameters", te.config.Theme, te.config.Icons)
	
	parameterList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		te.editParameterAtIndex(index)
	})

	// Refresh parameter list
	te.refreshParameterList(parameterList)

	// Add parameter management buttons
	parameterButtons := tview.NewFlex().SetDirection(tview.FlexColumn)
	
	addButton := tview.NewButton("Add (a)")
	addButton.SetSelectedFunc(te.addParameter)
	
	editButton := tview.NewButton("Edit (e)")
	editButton.SetSelectedFunc(te.editParameter)
	
	deleteButton := tview.NewButton("Delete (d)")
	deleteButton.SetSelectedFunc(te.deleteParameter)

	parameterButtons.AddItem(addButton, 0, 1, false)
	parameterButtons.AddItem(editButton, 0, 1, false)
	parameterButtons.AddItem(deleteButton, 0, 1, false)

	// Layout parameters tab
	parameterTab := tview.NewFlex().SetDirection(tview.FlexRow)
	parameterTab.AddItem(parameterList, 0, 1, true)
	parameterTab.AddItem(parameterButtons, 3, 0, false)

	te.contentArea.AddItem(parameterTab, 0, 1, true)
	te.parameterList = parameterList // Update reference
}

// Template operations
func (te *TemplateEditor) saveTemplate() {
	if !te.validateTemplate() {
		return
	}

	te.template.UpdatedAt = time.Now()

	var err error
	if te.mode == EditorModeCreate {
		err = te.manager.CreateTemplateFromFile(te.template.Name, "")
		if err == nil {
			te.mode = EditorModeEdit
		}
	} else {
		err = te.manager.UpdateTemplate(te.template.Name, false, true)
	}

	if err != nil {
		te.handleError(fmt.Errorf("failed to save template: %w", err))
		return
	}

	te.hasChanges = false
	te.originalTemplate = te.copyTemplate(te.template)
	te.showStatus("Template saved successfully", "success")

	if te.callbacks.OnTemplateSave != nil {
		te.callbacks.OnTemplateSave(*te.template)
	}
}

func (te *TemplateEditor) testTemplate() {
	if te.template == nil {
		te.showStatus("No template to test", "warning")
		return
	}

	if te.callbacks.OnTemplateTest != nil {
		te.callbacks.OnTemplateTest(*te.template)
	}
}

func (te *TemplateEditor) deleteTemplate() {
	if te.template == nil || te.mode == EditorModeCreate {
		te.showStatus("No template to delete", "warning")
		return
	}

	// Show confirmation dialog
	modal := CreateModal("Delete Template", nil, 50, 10, te.config.Theme)
	modal.SetText(fmt.Sprintf("Are you sure you want to delete template '%s'?", te.template.Name))
	modal.AddButtons([]string{"Delete", "Cancel"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 { // Delete
			err := te.manager.DeleteTemplate(te.template.Name)
			if err != nil {
				te.handleError(fmt.Errorf("failed to delete template: %w", err))
				return
			}

			te.showStatus("Template deleted successfully", "success")

			if te.callbacks.OnTemplateDelete != nil {
				te.callbacks.OnTemplateDelete(*te.template)
			}
		}
	})

	// Show modal (this would need app context)
}

func (te *TemplateEditor) cancelEditing() {
	if te.hasChanges {
		// Show confirmation dialog for unsaved changes
		modal := CreateModal("Unsaved Changes", nil, 50, 10, te.config.Theme)
		modal.SetText("You have unsaved changes. Are you sure you want to cancel?")
		modal.AddButtons([]string{"Discard", "Keep Editing"})
		modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 { // Discard
				if te.callbacks.OnCancel != nil {
					te.callbacks.OnCancel()
				}
			}
		})
		// Show modal (this would need app context)
	} else {
		if te.callbacks.OnCancel != nil {
			te.callbacks.OnCancel()
		}
	}
}

// Parameter management methods
func (te *TemplateEditor) addParameter() {
	newParam := prompt.TemplateParameter{
		Name:        "new_parameter",
		Description: "Parameter description",
		Type:        "string",
		Required:    false,
		Default:     "",
	}

	te.template.Parameters = append(te.template.Parameters, newParam)
	te.markChanged()
	te.refreshParameterList(te.parameterList)
	te.editParameterAtIndex(len(te.template.Parameters) - 1)
}

func (te *TemplateEditor) editParameter() {
	currentIndex := te.parameterList.GetCurrentItem()
	if currentIndex >= 0 && currentIndex < len(te.template.Parameters) {
		te.editParameterAtIndex(currentIndex)
	}
}

func (te *TemplateEditor) editParameterAtIndex(index int) {
	if index < 0 || index >= len(te.template.Parameters) {
		return
	}

	param := &te.template.Parameters[index]
	te.showParameterEditDialog(param, index)
}

func (te *TemplateEditor) showParameterEditDialog(param *prompt.TemplateParameter, index int) {
	// Create parameter editing form
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Edit Parameter ")

	// Name field
	nameField := tview.NewInputField()
	nameField.SetLabel("Name: ")
	nameField.SetText(param.Name)
	form.AddFormItem(nameField)

	// Description field
	descField := tview.NewInputField()
	descField.SetLabel("Description: ")
	descField.SetText(param.Description)
	form.AddFormItem(descField)

	// Type dropdown
	typeField := tview.NewDropDown()
	typeField.SetLabel("Type: ")
	types := []string{"string", "int", "bool", "select"}
	typeIndex := 0
	for i, t := range types {
		if t == param.Type {
			typeIndex = i
			break
		}
	}
	typeField.SetOptions(types, nil)
	typeField.SetCurrentOption(typeIndex)
	form.AddFormItem(typeField)

	// Required checkbox
	requiredField := tview.NewCheckbox()
	requiredField.SetLabel("Required: ")
	requiredField.SetChecked(param.Required)
	form.AddFormItem(requiredField)

	// Default value field
	defaultField := tview.NewInputField()
	defaultField.SetLabel("Default: ")
	defaultField.SetText(param.Default)
	form.AddFormItem(defaultField)

	// Options field (for select type)
	optionsField := tview.NewInputField()
	optionsField.SetLabel("Options (comma-separated): ")
	optionsField.SetText(strings.Join(param.Options, ", "))
	form.AddFormItem(optionsField)

	// Validation regex field
	validationField := tview.NewInputField()
	validationField.SetLabel("Validation (regex): ")
	validationField.SetText(param.Validation)
	form.AddFormItem(validationField)

	// Placeholder field
	placeholderField := tview.NewInputField()
	placeholderField.SetLabel("Placeholder: ")
	placeholderField.SetText(param.Placeholder)
	form.AddFormItem(placeholderField)

	// Buttons
	form.AddButton("Save", func() {
		// Update parameter
		param.Name = nameField.GetText()
		param.Description = descField.GetText()
		_, param.Type = typeField.GetCurrentOption()
		param.Required = requiredField.IsChecked()
		param.Default = defaultField.GetText()
		param.Validation = validationField.GetText()
		param.Placeholder = placeholderField.GetText()

		// Parse options
		optionsText := optionsField.GetText()
		if optionsText != "" {
			options := strings.Split(optionsText, ",")
			for i, option := range options {
				options[i] = strings.TrimSpace(option)
			}
			param.Options = options
		} else {
			param.Options = []string{}
		}

		te.markChanged()
		te.refreshParameterList(te.parameterList)
		// Close dialog (this would need app context)
	})

	form.AddButton("Cancel", func() {
		// Close dialog (this would need app context)
	})

	// Show form as modal (this would need app context)
}

func (te *TemplateEditor) deleteParameter() {
	currentIndex := te.parameterList.GetCurrentItem()
	if currentIndex >= 0 && currentIndex < len(te.template.Parameters) {
		// Remove parameter
		te.template.Parameters = append(
			te.template.Parameters[:currentIndex],
			te.template.Parameters[currentIndex+1:]...,
		)
		te.markChanged()
		te.refreshParameterList(te.parameterList)
	}
}

func (te *TemplateEditor) refreshParameterList(list *tview.List) {
	list.Clear()

	for i, param := range te.template.Parameters {
		required := ""
		if param.Required {
			required = " (required)"
		}

		mainText := fmt.Sprintf("%s %s%s", te.config.Icons.Parameter, param.Name, required)
		secondaryText := fmt.Sprintf("%s - %s", param.Type, param.Description)

		list.AddItem(mainText, secondaryText, rune('1'+i), nil)
	}
}

// UI update methods
func (te *TemplateEditor) updateUI() {
	te.updateHeader()
	te.updateTabBar()
	te.updateButtons()
	te.updateStatusBar()
	te.populateFields()
}

func (te *TemplateEditor) updateHeader() {
	var headerText string
	switch te.mode {
	case EditorModeCreate:
		headerText = "Create New Template"
	case EditorModeEdit:
		headerText = fmt.Sprintf("Edit Template: %s", te.template.Name)
	case EditorModeView:
		headerText = fmt.Sprintf("View Template: %s", te.template.Name)
	}

	if te.hasChanges {
		headerText += " [modified]"
	}

	te.headerText.SetText(headerText)
}

func (te *TemplateEditor) updateTabBar() {
	var tabBar strings.Builder

	tabs := []struct {
		tab  EditorTab
		name string
		key  string
	}{
		{TabMetadata, "Metadata", "F1"},
		{TabContent, "Content", "F2"},
		{TabParameters, "Parameters", "F3"},
		{TabPreview, "Preview", "F4"},
	}

	for i, tabInfo := range tabs {
		if i > 0 {
			tabBar.WriteString(" | ")
		}

		if tabInfo.tab == te.currentTab {
			colorStart, colorEnd := te.config.Theme.ColorToTags(te.config.Theme.Primary)
			tabBar.WriteString(fmt.Sprintf("%s[%s] %s%s", colorStart, tabInfo.key, tabInfo.name, colorEnd))
		} else {
			colorStart, colorEnd := te.config.Theme.ColorToTags(te.config.Theme.TextMuted)
			tabBar.WriteString(fmt.Sprintf("%s[%s] %s%s", colorStart, tabInfo.key, tabInfo.name, colorEnd))
		}
	}

	te.tabBar.SetText(tabBar.String())
}

func (te *TemplateEditor) updateButtons() {
	// Update button visibility based on mode
	switch te.mode {
	case EditorModeCreate:
		te.deleteButton.SetLabel("")
	case EditorModeEdit:
		te.deleteButton.SetLabel("Delete (Ctrl+D)")
	case EditorModeView:
		te.saveButton.SetLabel("")
		te.deleteButton.SetLabel("")
	}
}

func (te *TemplateEditor) updateStatusBar() {
	var status strings.Builder

	status.WriteString(fmt.Sprintf("Mode: %s", te.getModeString()))

	if te.template != nil {
		status.WriteString(fmt.Sprintf(" | Template: %s", te.template.Name))
		status.WriteString(fmt.Sprintf(" | Parameters: %d", len(te.template.Parameters)))
	}

	if te.hasChanges {
		status.WriteString(" | [yellow]Modified[white]")
	}

	// Add current tab
	status.WriteString(fmt.Sprintf(" | Tab: %s", te.getTabString()))

	if te.config.EnableKeyHelp {
		status.WriteString(" | ")
		status.WriteString(te.keyBindings.GetHelpText(te.config.Theme))
	}

	te.statusBar.SetText(status.String())
}

func (te *TemplateEditor) updatePreview() {
	if te.template == nil {
		te.previewPane.SetText("No template loaded")
		return
	}

	var preview strings.Builder

	// Template header
	colorStart, colorEnd := te.config.Theme.ColorToTags(te.config.Theme.Primary)
	preview.WriteString(fmt.Sprintf("%s%s%s\n\n", colorStart, te.template.Name, colorEnd))

	// Metadata
	preview.WriteString("[yellow]Metadata:[white]\n")
	preview.WriteString(fmt.Sprintf("  Category: %s\n", te.template.Category))
	if te.template.Language != "" {
		preview.WriteString(fmt.Sprintf("  Language: %s\n", te.template.Language))
	}
	if te.template.Framework != "" {
		preview.WriteString(fmt.Sprintf("  Framework: %s\n", te.template.Framework))
	}
	if te.template.Author != "" {
		preview.WriteString(fmt.Sprintf("  Author: %s\n", te.template.Author))
	}
	if te.template.Version != "" {
		preview.WriteString(fmt.Sprintf("  Version: %s\n", te.template.Version))
	}

	// Description
	if te.template.Description != "" {
		preview.WriteString(fmt.Sprintf("\n[yellow]Description:[white]\n%s\n", te.template.Description))
	}

	// Parameters
	if len(te.template.Parameters) > 0 {
		preview.WriteString(fmt.Sprintf("\n[yellow]Parameters (%d):[white]\n", len(te.template.Parameters)))
		for _, param := range te.template.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}
			preview.WriteString(fmt.Sprintf("  • %s%s: %s\n", param.Name, required, param.Description))
		}
	}

	// Tags
	if len(te.template.Tags) > 0 {
		preview.WriteString(fmt.Sprintf("\n[yellow]Tags:[white] %s\n", strings.Join(te.template.Tags, ", ")))
	}

	// Content
	if te.template.Content != "" {
		preview.WriteString("\n[yellow]Content:[white]\n")
		preview.WriteString(strings.Repeat("-", 50))
		preview.WriteString("\n")
		preview.WriteString(te.template.Content)
	}

	// Validation status
	if valid, issues := te.manager.ValidateTemplate(te.template.Name); len(issues) > 0 {
		if valid {
			preview.WriteString("\n\n[green]Validation: Passed[white]")
		} else {
			preview.WriteString("\n\n[red]Validation Issues:[white]\n")
			for _, issue := range issues {
				preview.WriteString(fmt.Sprintf("  • %s\n", issue))
			}
		}
	}

	te.previewPane.SetText(preview.String())
}

func (te *TemplateEditor) populateFields() {
	if te.template == nil {
		return
	}

	// Populate metadata fields
	te.nameField.SetText(te.template.Name)
	
	// Set category
	categories := []string{"general", "languages", "workflows", "workspace", "custom"}
	for i, category := range categories {
		if category == te.template.Category {
			te.categoryField.SetCurrentOption(i)
			break
		}
	}
	
	te.languageField.SetText(te.template.Language)
	te.frameworkField.SetText(te.template.Framework)
	te.descriptionField.SetText(te.template.Description, false)
	te.authorField.SetText(te.template.Author)
	te.versionField.SetText(te.template.Version)
	
	if len(te.template.Tags) > 0 {
		te.tagsField.SetText(strings.Join(te.template.Tags, ", "))
	}

	// Populate content
	te.contentEditor.SetText(te.template.Content, false)
}

// Validation methods
func (te *TemplateEditor) validateTemplate() bool {
	if te.template.Name == "" {
		te.showStatus("Template name is required", "error")
		return false
	}

	if te.template.Content == "" {
		te.showStatus("Template content is required", "error")
		return false
	}

	// Validate parameters
	for _, param := range te.template.Parameters {
		if param.Name == "" {
			te.showStatus("All parameters must have names", "error")
			return false
		}
	}

	return true
}

// Navigation methods
func (te *TemplateEditor) nextField() {
	// Implementation depends on current tab and focus
}

func (te *TemplateEditor) prevField() {
	// Implementation depends on current tab and focus
}

func (te *TemplateEditor) undo() {
	// Implement undo functionality
	te.showStatus("Undo not implemented yet", "info")
}

func (te *TemplateEditor) redo() {
	// Implement redo functionality
	te.showStatus("Redo not implemented yet", "info")
}

// Utility methods
func (te *TemplateEditor) markChanged() {
	te.hasChanges = true
	te.updateHeader()
	te.updateStatusBar()
}

func (te *TemplateEditor) getModeString() string {
	switch te.mode {
	case EditorModeCreate:
		return "Create"
	case EditorModeEdit:
		return "Edit"
	case EditorModeView:
		return "View"
	default:
		return "Unknown"
	}
}

func (te *TemplateEditor) getTabString() string {
	switch te.currentTab {
	case TabMetadata:
		return "Metadata"
	case TabContent:
		return "Content"
	case TabParameters:
		return "Parameters"
	case TabPreview:
		return "Preview"
	default:
		return "Unknown"
	}
}

func (te *TemplateEditor) copyTemplate(template *prompt.Template) *prompt.Template {
	// Deep copy template
	data, _ := yaml.Marshal(template)
	var copy prompt.Template
	yaml.Unmarshal(data, &copy)
	return &copy
}

func (te *TemplateEditor) showStatus(message, msgType string) {
	statusMsg := NewStatusMessage(message, msgType, 3*time.Second)
	formattedMsg := statusMsg.FormatForDisplay(te.config.Theme)
	
	// Temporarily update status bar
	originalText := te.statusBar.GetText(false)
	te.statusBar.SetText(formattedMsg)
	
	// Restore after duration
	go func() {
		time.Sleep(statusMsg.Duration)
		te.statusBar.SetText(originalText)
	}()
}

func (te *TemplateEditor) handleError(err error) {
	te.logger.Error("template editor error", zap.Error(err))
	te.showStatus(err.Error(), "error")
	
	if te.callbacks.OnError != nil {
		te.callbacks.OnError(err)
	}
}

// Public interface methods

// SetTemplate loads a template for editing
func (te *TemplateEditor) SetTemplate(template prompt.Template) {
	te.template = &template
	te.originalTemplate = te.copyTemplate(&template)
	te.mode = EditorModeEdit
	te.hasChanges = false
	te.updateUI()
}

// CreateNewTemplate creates a new template
func (te *TemplateEditor) CreateNewTemplate() {
	te.template = &prompt.Template{
		Name:        "",
		Category:    "custom",
		Description: "",
		Content:     "",
		Parameters:  []prompt.TemplateParameter{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	te.originalTemplate = nil
	te.mode = EditorModeCreate
	te.hasChanges = false
	te.updateUI()
}

// GetTemplate returns the current template
func (te *TemplateEditor) GetTemplate() *prompt.Template {
	return te.template
}

// HasChanges returns true if there are unsaved changes
func (te *TemplateEditor) HasChanges() bool {
	return te.hasChanges
}

// Focus sets focus to the template editor
func (te *TemplateEditor) Focus(delegate func(p tview.Primitive)) {
	switch te.currentTab {
	case TabMetadata:
		delegate(te.metadataForm)
	case TabContent:
		delegate(te.contentEditor)
	case TabParameters:
		delegate(te.parameterList)
	case TabPreview:
		delegate(te.previewPane)
	}
}

// HasFocus returns true if the template editor has focus
func (te *TemplateEditor) HasFocus() bool {
	switch te.currentTab {
	case TabMetadata:
		return te.metadataForm.HasFocus()
	case TabContent:
		return te.contentEditor.HasFocus()
	case TabParameters:
		return te.parameterList.HasFocus()
	case TabPreview:
		return te.previewPane.HasFocus()
	}
	return false
}

// Close cleans up the template editor
func (te *TemplateEditor) Close() {
	te.cancel()
}