package components

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/prompt"
)

// GeneratorStep represents a step in the generation wizard
type GeneratorStep int

const (
	StepSelectTemplate GeneratorStep = iota
	StepCollectParameters
	StepReviewAndGenerate
	StepShowResult
)

// ParameterInput represents a parameter input field
type ParameterInput struct {
	Parameter   prompt.TemplateParameter
	InputField  *tview.InputField
	DropDown    *tview.DropDown
	CheckBox    *tview.Checkbox
	Value       string
	IsValid     bool
	ErrorMsg    string
}

// GenerationState holds the current generation state
type GenerationState struct {
	Step         GeneratorStep
	Template     *prompt.Template
	Parameters   map[string]*ParameterInput
	Context      *prompt.WorkspaceContext
	Result       *prompt.GenerationResult
	ErrorMessage string
}

// TemplateGeneratorCallbacks defines callback functions for generator events
type TemplateGeneratorCallbacks struct {
	OnGenerationComplete func(result *prompt.GenerationResult)
	OnError              func(err error)
	OnCancel             func()
}

// TemplateGenerator provides an interactive TUI for generating prompts from templates
type TemplateGenerator struct {
	*tview.Flex

	// Configuration
	config    *PromptUIConfig
	callbacks *TemplateGeneratorCallbacks
	manager   prompt.Manager
	logger    *zap.Logger

	// UI components
	headerText     *tview.TextView
	stepIndicator  *tview.TextView
	contentArea    *tview.Flex
	buttonBar      *tview.Flex
	statusBar      *tview.TextView

	// Step-specific components
	templateBrowser *PromptBrowser
	parameterForm   *tview.Form
	reviewPane      *tview.TextView
	resultPane      *tview.TextView

	// Buttons
	prevButton     *tview.Button
	nextButton     *tview.Button
	generateButton *tview.Button
	cancelButton   *tview.Button
	copyButton     *tview.Button
	saveButton     *tview.Button

	// State
	state       *GenerationState
	keyBindings *KeyBindings

	// Context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTemplateGenerator creates a new template generator component
func NewTemplateGenerator(manager prompt.Manager, config *PromptUIConfig, callbacks *TemplateGeneratorCallbacks, logger *zap.Logger) *TemplateGenerator {
	if config == nil {
		config = DefaultPromptUIConfig()
	}

	if callbacks == nil {
		callbacks = &TemplateGeneratorCallbacks{}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	tg := &TemplateGenerator{
		Flex:      tview.NewFlex(),
		config:    config,
		callbacks: callbacks,
		manager:   manager,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		state: &GenerationState{
			Step:       StepSelectTemplate,
			Parameters: make(map[string]*ParameterInput),
		},
	}

	tg.initKeyBindings()
	tg.initUI()
	tg.initWorkspaceContext()

	return tg
}

// initKeyBindings sets up keyboard shortcuts
func (tg *TemplateGenerator) initKeyBindings() {
	tg.keyBindings = NewKeyBindings()

	// Navigation
	tg.keyBindings.AddKey(tcell.KeyTab, "Next Field", func() { tg.nextField() })
	tg.keyBindings.AddKey(tcell.KeyBacktab, "Prev Field", func() { tg.prevField() })

	// Step navigation
	tg.keyBindings.AddKey(tcell.KeyF1, "Previous Step", func() { tg.previousStep() })
	tg.keyBindings.AddKey(tcell.KeyF2, "Next Step", func() { tg.nextStep() })
	tg.keyBindings.AddKey(tcell.KeyF3, "Generate", func() { tg.generate() })

	// Actions
	tg.keyBindings.AddRune('g', "Generate", func() { tg.generate() })
	tg.keyBindings.AddRune('c', "Copy Result", func() { tg.copyResult() })
	tg.keyBindings.AddRune('s', "Save Result", func() { tg.saveResult() })
	tg.keyBindings.AddKey(tcell.KeyEsc, "Cancel", func() { tg.cancelGeneration() })

	// Validation
	tg.keyBindings.AddRune('v', "Validate", func() { tg.validateCurrentStep() })
	tg.keyBindings.AddKey(tcell.KeyF5, "Refresh Context", func() { tg.refreshWorkspaceContext() })
}

// initUI initializes the user interface components
func (tg *TemplateGenerator) initUI() {
	tg.SetDirection(tview.FlexRow)

	// Create header
	tg.headerText = tview.NewTextView()
	tg.headerText.SetBorder(true)
	tg.headerText.SetTitle(fmt.Sprintf(" %s Template Generator ", tg.config.Icons.Generate))
	tg.headerText.SetTitleAlign(tview.AlignCenter)
	tg.headerText.SetDynamicColors(true)
	tg.headerText.SetTextAlign(tview.AlignCenter)

	// Create step indicator
	tg.stepIndicator = tview.NewTextView()
	tg.stepIndicator.SetDynamicColors(true)
	tg.stepIndicator.SetTextAlign(tview.AlignCenter)

	// Create content area
	tg.contentArea = tview.NewFlex()
	tg.contentArea.SetBorder(true)
	tg.contentArea.SetTitle(" Content ")

	// Create button bar
	tg.buttonBar = tview.NewFlex().SetDirection(tview.FlexColumn)
	tg.initButtons()

	// Create status bar
	tg.statusBar = tview.NewTextView()
	tg.statusBar.SetDynamicColors(true)
	tg.statusBar.SetTextAlign(tview.AlignLeft)

	// Layout
	tg.AddItem(tg.headerText, 3, 0, false)
	tg.AddItem(tg.stepIndicator, 1, 0, false)
	tg.AddItem(tg.contentArea, 0, 1, true)
	tg.AddItem(tg.buttonBar, 3, 0, false)
	tg.AddItem(tg.statusBar, 1, 0, false)

	// Initialize first step
	tg.setupStepSelectTemplate()
	tg.updateUI()

	// Set up input capture
	tg.SetInputCapture(tg.handleKeyPress)
}

// initButtons creates and configures buttons
func (tg *TemplateGenerator) initButtons() {
	// Previous button
	tg.prevButton = tview.NewButton("Previous (F1)")
	tg.prevButton.SetSelectedFunc(tg.previousStep)

	// Next button
	tg.nextButton = tview.NewButton("Next (F2)")
	tg.nextButton.SetSelectedFunc(tg.nextStep)

	// Generate button
	tg.generateButton = tview.NewButton("Generate (F3)")
	tg.generateButton.SetSelectedFunc(tg.generate)

	// Cancel button
	tg.cancelButton = tview.NewButton("Cancel (Esc)")
	tg.cancelButton.SetSelectedFunc(tg.cancelGeneration)

	// Copy button
	tg.copyButton = tview.NewButton("Copy (c)")
	tg.copyButton.SetSelectedFunc(tg.copyResult)

	// Save button
	tg.saveButton = tview.NewButton("Save (s)")
	tg.saveButton.SetSelectedFunc(tg.saveResult)

	// Add buttons to bar (will be shown/hidden based on step)
	tg.buttonBar.AddItem(tg.prevButton, 0, 1, false)
	tg.buttonBar.AddItem(tg.nextButton, 0, 1, false)
	tg.buttonBar.AddItem(tg.generateButton, 0, 1, false)
	tg.buttonBar.AddItem(tg.cancelButton, 0, 1, false)
	tg.buttonBar.AddItem(tg.copyButton, 0, 1, false)
	tg.buttonBar.AddItem(tg.saveButton, 0, 1, false)
}

// handleKeyPress processes keyboard input
func (tg *TemplateGenerator) handleKeyPress(event *tcell.EventKey) *tcell.EventKey {
	// Try key bindings first
	if tg.keyBindings.Handle(event) {
		return nil
	}

	return event
}

// Step setup methods
func (tg *TemplateGenerator) setupStepSelectTemplate() {
	tg.contentArea.Clear()

	// Create template browser for selection
	browserCallbacks := &PromptBrowserCallbacks{
		OnTemplateSelect: func(template prompt.Template) {
			tg.state.Template = &template
			tg.updateUI()
		},
		OnTemplateGenerate: func(template prompt.Template) {
			tg.state.Template = &template
			tg.nextStep()
		},
		OnError: func(err error) {
			tg.handleError(err)
		},
	}

	tg.templateBrowser = NewPromptBrowser(tg.manager, tg.config, browserCallbacks, tg.logger)
	tg.contentArea.AddItem(tg.templateBrowser, 0, 1, true)
}

func (tg *TemplateGenerator) setupStepCollectParameters() {
	tg.contentArea.Clear()

	if tg.state.Template == nil {
		tg.showError("No template selected")
		return
	}

	// Create parameter form
	tg.parameterForm = tview.NewForm()
	tg.parameterForm.SetBorder(false)

	// Clear existing parameters
	tg.state.Parameters = make(map[string]*ParameterInput)

	for _, param := range tg.state.Template.Parameters {
		paramInput := &ParameterInput{
			Parameter: param,
			IsValid:   !param.Required, // Optional parameters are valid by default
		}

		switch param.Type {
		case "string", "":
			// Text input field
			paramInput.InputField = tview.NewInputField()
			paramInput.InputField.SetLabel(fmt.Sprintf("%s: ", param.Name))
			if param.Placeholder != "" {
				paramInput.InputField.SetPlaceholder(param.Placeholder)
			}
			if param.Default != "" {
				paramInput.InputField.SetText(param.Default)
				paramInput.Value = param.Default
			}

			// Set up validation
			paramInput.InputField.SetChangedFunc(func(text string) {
				paramInput.Value = text
				paramInput.IsValid = tg.validateParameter(paramInput)
				tg.updateParameterValidation(paramInput)
			})

			tg.parameterForm.AddFormItem(paramInput.InputField)

		case "select":
			// Dropdown
			paramInput.DropDown = tview.NewDropDown()
			paramInput.DropDown.SetLabel(fmt.Sprintf("%s: ", param.Name))
			
			options := param.Options
			if len(options) == 0 {
				options = []string{"Option 1", "Option 2", "Option 3"}
			}
			
			paramInput.DropDown.SetOptions(options, func(option string, optionIndex int) {
				paramInput.Value = option
				paramInput.IsValid = true
				tg.updateParameterValidation(paramInput)
			})

			// Set default if specified
			if param.Default != "" {
				for i, option := range options {
					if option == param.Default {
						paramInput.DropDown.SetCurrentOption(i)
						paramInput.Value = param.Default
						break
					}
				}
			}

			tg.parameterForm.AddFormItem(paramInput.DropDown)

		case "bool":
			// Checkbox
			paramInput.CheckBox = tview.NewCheckbox()
			paramInput.CheckBox.SetLabel(fmt.Sprintf("%s: ", param.Name))
			
			if param.Default == "true" {
				paramInput.CheckBox.SetChecked(true)
				paramInput.Value = "true"
			} else {
				paramInput.Value = "false"
			}

			paramInput.CheckBox.SetChangedFunc(func(checked bool) {
				paramInput.Value = strconv.FormatBool(checked)
				paramInput.IsValid = true
				tg.updateParameterValidation(paramInput)
			})

			tg.parameterForm.AddFormItem(paramInput.CheckBox)

		default:
			// Default to text input
			paramInput.InputField = tview.NewInputField()
			paramInput.InputField.SetLabel(fmt.Sprintf("%s (%s): ", param.Name, param.Type))
			if param.Default != "" {
				paramInput.InputField.SetText(param.Default)
				paramInput.Value = param.Default
			}

			paramInput.InputField.SetChangedFunc(func(text string) {
				paramInput.Value = text
				paramInput.IsValid = tg.validateParameter(paramInput)
				tg.updateParameterValidation(paramInput)
			})

			tg.parameterForm.AddFormItem(paramInput.InputField)
		}

		tg.state.Parameters[param.Name] = paramInput
	}

	// Add parameter descriptions
	descriptionPane := tview.NewTextView()
	descriptionPane.SetBorder(true)
	descriptionPane.SetTitle(" Parameter Information ")
	descriptionPane.SetDynamicColors(true)
	descriptionPane.SetWordWrap(true)

	var descriptions strings.Builder
	for _, param := range tg.state.Template.Parameters {
		required := ""
		if param.Required {
			required = " [red](required)[white]"
		}

		descriptions.WriteString(fmt.Sprintf("[yellow]%s[white]%s\n", param.Name, required))
		descriptions.WriteString(fmt.Sprintf("  %s\n", param.Description))
		if param.Default != "" {
			descriptions.WriteString(fmt.Sprintf("  Default: %s\n", param.Default))
		}
		descriptions.WriteString("\n")
	}

	descriptionPane.SetText(descriptions.String())

	// Layout: form on left, descriptions on right
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentFlex.AddItem(tg.parameterForm, 0, 1, true)
	contentFlex.AddItem(descriptionPane, 0, 1, false)

	tg.contentArea.AddItem(contentFlex, 0, 1, true)
}

func (tg *TemplateGenerator) setupStepReviewAndGenerate() {
	tg.contentArea.Clear()

	// Create review pane
	tg.reviewPane = tview.NewTextView()
	tg.reviewPane.SetBorder(true)
	tg.reviewPane.SetTitle(" Review Configuration ")
	tg.reviewPane.SetDynamicColors(true)
	tg.reviewPane.SetWordWrap(true)
	tg.reviewPane.SetScrollable(true)

	tg.updateReviewPane()
	tg.contentArea.AddItem(tg.reviewPane, 0, 1, true)
}

func (tg *TemplateGenerator) setupStepShowResult() {
	tg.contentArea.Clear()

	// Create result pane
	tg.resultPane = tview.NewTextView()
	tg.resultPane.SetBorder(true)
	tg.resultPane.SetTitle(" Generated Prompt ")
	tg.resultPane.SetDynamicColors(true)
	tg.resultPane.SetWordWrap(true)
	tg.resultPane.SetScrollable(true)

	if tg.state.Result != nil {
		var result strings.Builder
		
		// Header with metadata
		result.WriteString("[yellow]Generated Prompt[white]\n")
		result.WriteString(fmt.Sprintf("Template: %s\n", tg.state.Result.Template.Name))
		result.WriteString(fmt.Sprintf("Generated: %s\n", FormatTime(tg.state.Result.GeneratedAt)))
		result.WriteString(fmt.Sprintf("Words: %d | Characters: %d\n", tg.state.Result.WordCount, tg.state.Result.CharCount))
		
		if tg.state.Result.ValidationStatus.Valid {
			result.WriteString(fmt.Sprintf("[green]Validation: Passed (Score: %d/100)[white]\n", tg.state.Result.ValidationStatus.Score))
		} else {
			result.WriteString("[red]Validation: Failed[white]\n")
			for _, issue := range tg.state.Result.ValidationStatus.Issues {
				result.WriteString(fmt.Sprintf("  • %s\n", issue))
			}
		}

		result.WriteString("\n[yellow]Content:[white]\n")
		result.WriteString(strings.Repeat("-", 50))
		result.WriteString("\n\n")
		result.WriteString(tg.state.Result.Content)
		
		tg.resultPane.SetText(result.String())
	} else if tg.state.ErrorMessage != "" {
		tg.resultPane.SetText(fmt.Sprintf("[red]Generation Failed:[white]\n\n%s", tg.state.ErrorMessage))
	}

	tg.contentArea.AddItem(tg.resultPane, 0, 1, true)
}

// Validation methods
func (tg *TemplateGenerator) validateParameter(paramInput *ParameterInput) bool {
	param := paramInput.Parameter
	value := paramInput.Value

	// Check if required parameter is empty
	if param.Required && value == "" {
		paramInput.ErrorMsg = "This parameter is required"
		return false
	}

	// Check validation pattern if specified
	if param.Validation != "" && value != "" {
		matched, err := regexp.MatchString(param.Validation, value)
		if err != nil {
			paramInput.ErrorMsg = "Invalid validation pattern"
			return false
		}
		if !matched {
			paramInput.ErrorMsg = "Value does not match required pattern"
			return false
		}
	}

	// Type-specific validation
	switch param.Type {
	case "int":
		if value != "" {
			if _, err := strconv.Atoi(value); err != nil {
				paramInput.ErrorMsg = "Value must be an integer"
				return false
			}
		}
	case "bool":
		if value != "" && value != "true" && value != "false" {
			paramInput.ErrorMsg = "Value must be true or false"
			return false
		}
	}

	paramInput.ErrorMsg = ""
	return true
}

func (tg *TemplateGenerator) updateParameterValidation(paramInput *ParameterInput) {
	// Update UI to show validation status
	if paramInput.InputField != nil {
		if paramInput.IsValid {
			paramInput.InputField.SetFieldTextColor(tg.config.Theme.TextPrimary)
		} else {
			paramInput.InputField.SetFieldTextColor(tg.config.Theme.Error)
		}
	}
}

func (tg *TemplateGenerator) validateCurrentStep() bool {
	switch tg.state.Step {
	case StepSelectTemplate:
		return tg.state.Template != nil

	case StepCollectParameters:
		allValid := true
		for _, paramInput := range tg.state.Parameters {
			if !paramInput.IsValid {
				allValid = false
			}
		}
		return allValid

	case StepReviewAndGenerate:
		return true

	case StepShowResult:
		return tg.state.Result != nil
	}

	return false
}

// Step navigation methods
func (tg *TemplateGenerator) nextStep() {
	if !tg.validateCurrentStep() {
		tg.showStatus("Please complete the current step before proceeding", "warning")
		return
	}

	switch tg.state.Step {
	case StepSelectTemplate:
		if tg.state.Template != nil {
			tg.state.Step = StepCollectParameters
			tg.setupStepCollectParameters()
		}

	case StepCollectParameters:
		tg.state.Step = StepReviewAndGenerate
		tg.setupStepReviewAndGenerate()

	case StepReviewAndGenerate:
		// Don't auto-advance, user must click generate
		return

	case StepShowResult:
		// Already at final step
		return
	}

	tg.updateUI()
}

func (tg *TemplateGenerator) previousStep() {
	switch tg.state.Step {
	case StepSelectTemplate:
		// Already at first step
		return

	case StepCollectParameters:
		tg.state.Step = StepSelectTemplate
		tg.setupStepSelectTemplate()

	case StepReviewAndGenerate:
		tg.state.Step = StepCollectParameters
		tg.setupStepCollectParameters()

	case StepShowResult:
		tg.state.Step = StepReviewAndGenerate
		tg.setupStepReviewAndGenerate()
	}

	tg.updateUI()
}

func (tg *TemplateGenerator) generate() {
	if tg.state.Template == nil {
		tg.showError("No template selected")
		return
	}

	// Collect parameters
	parameters := make(map[string]string)
	for name, paramInput := range tg.state.Parameters {
		if !paramInput.IsValid {
			tg.showError(fmt.Sprintf("Invalid parameter '%s': %s", name, paramInput.ErrorMsg))
			return
		}
		parameters[name] = paramInput.Value
	}

	// Create generation config
	config := &prompt.GenerationConfig{
		TemplateName: tg.state.Template.Name,
		Interactive:  false,
		Context:      tg.state.Context,
		Parameters:   parameters,
		Validate:     true,
		OutputFormat: "markdown",
	}

	// Generate prompt
	tg.showStatus("Generating prompt...", "info")
	result, err := tg.manager.GeneratePrompt(config)
	if err != nil {
		tg.state.ErrorMessage = err.Error()
		tg.showError(fmt.Sprintf("Generation failed: %v", err))
	} else {
		tg.state.Result = result
		tg.showStatus("Prompt generated successfully!", "success")
	}

	// Move to result step
	tg.state.Step = StepShowResult
	tg.setupStepShowResult()
	tg.updateUI()

	// Call completion callback
	if tg.callbacks.OnGenerationComplete != nil && result != nil {
		tg.callbacks.OnGenerationComplete(result)
	}
}

// Action methods
func (tg *TemplateGenerator) copyResult() {
	if tg.state.Result == nil {
		tg.showStatus("No result to copy", "warning")
		return
	}

	err := tg.manager.CopyToClipboard(tg.state.Result.Content)
	if err != nil {
		tg.handleError(fmt.Errorf("failed to copy result: %w", err))
		return
	}

	tg.showStatus("Result copied to clipboard", "success")
}

func (tg *TemplateGenerator) saveResult() {
	if tg.state.Result == nil {
		tg.showStatus("No result to save", "warning")
		return
	}

	// This would typically show a file dialog
	// For now, we'll save to a default location
	filename := fmt.Sprintf("prompt_%s_%d.md", 
		strings.ReplaceAll(tg.state.Template.Name, " ", "_"),
		time.Now().Unix())

	err := tg.manager.SaveToFile(tg.state.Result.Content, filename)
	if err != nil {
		tg.handleError(fmt.Errorf("failed to save result: %w", err))
		return
	}

	tg.showStatus(fmt.Sprintf("Result saved to %s", filename), "success")
}

func (tg *TemplateGenerator) cancelGeneration() {
	if tg.callbacks.OnCancel != nil {
		tg.callbacks.OnCancel()
	}
}

func (tg *TemplateGenerator) nextField() {
	// Move to next form field
	app := tg.GetApplication()
	if app != nil && tg.parameterForm != nil {
		app.SetFocus(tg.parameterForm)
	}
}

func (tg *TemplateGenerator) prevField() {
	// Move to previous form field
	app := tg.GetApplication()
	if app != nil && tg.parameterForm != nil {
		app.SetFocus(tg.parameterForm)
	}
}

// UI update methods
func (tg *TemplateGenerator) updateUI() {
	tg.updateHeader()
	tg.updateStepIndicator()
	tg.updateButtons()
	tg.updateStatusBar()
}

func (tg *TemplateGenerator) updateHeader() {
	var headerText string
	
	switch tg.state.Step {
	case StepSelectTemplate:
		headerText = "Step 1: Select a Template"
	case StepCollectParameters:
		headerText = fmt.Sprintf("Step 2: Configure Parameters for '%s'", tg.state.Template.Name)
	case StepReviewAndGenerate:
		headerText = "Step 3: Review Configuration and Generate"
	case StepShowResult:
		headerText = "Step 4: Generated Prompt Result"
	}

	tg.headerText.SetText(headerText)
}

func (tg *TemplateGenerator) updateStepIndicator() {
	var indicator strings.Builder
	
	steps := []string{"Select", "Configure", "Review", "Result"}
	for i, stepName := range steps {
		if i > 0 {
			indicator.WriteString(" → ")
		}

		if GeneratorStep(i) == tg.state.Step {
			colorStart, colorEnd := tg.config.Theme.ColorToTags(tg.config.Theme.Primary)
			indicator.WriteString(fmt.Sprintf("%s[%s]%s", colorStart, stepName, colorEnd))
		} else if GeneratorStep(i) < tg.state.Step {
			colorStart, colorEnd := tg.config.Theme.ColorToTags(tg.config.Theme.Success)
			indicator.WriteString(fmt.Sprintf("%s%s%s", colorStart, stepName, colorEnd))
		} else {
			colorStart, colorEnd := tg.config.Theme.ColorToTags(tg.config.Theme.TextMuted)
			indicator.WriteString(fmt.Sprintf("%s%s%s", colorStart, stepName, colorEnd))
		}
	}

	tg.stepIndicator.SetText(indicator.String())
}

func (tg *TemplateGenerator) updateButtons() {
	// Hide all buttons first
	tg.prevButton.SetLabel("")
	tg.nextButton.SetLabel("")
	tg.generateButton.SetLabel("")
	tg.copyButton.SetLabel("")
	tg.saveButton.SetLabel("")

	switch tg.state.Step {
	case StepSelectTemplate:
		tg.nextButton.SetLabel("Next (F2)")
		tg.cancelButton.SetLabel("Cancel (Esc)")

	case StepCollectParameters:
		tg.prevButton.SetLabel("Previous (F1)")
		tg.nextButton.SetLabel("Next (F2)")
		tg.cancelButton.SetLabel("Cancel (Esc)")

	case StepReviewAndGenerate:
		tg.prevButton.SetLabel("Previous (F1)")
		tg.generateButton.SetLabel("Generate (F3)")
		tg.cancelButton.SetLabel("Cancel (Esc)")

	case StepShowResult:
		tg.prevButton.SetLabel("Previous (F1)")
		tg.copyButton.SetLabel("Copy (c)")
		tg.saveButton.SetLabel("Save (s)")
		tg.cancelButton.SetLabel("Done (Esc)")
	}
}

func (tg *TemplateGenerator) updateStatusBar() {
	var status strings.Builder

	if tg.state.Template != nil {
		status.WriteString(fmt.Sprintf("Template: %s", tg.state.Template.Name))
	}

	if tg.state.Context != nil {
		if status.Len() > 0 {
			status.WriteString(" | ")
		}
		status.WriteString(fmt.Sprintf("Context: %s", tg.state.Context.Language))
	}

	if len(tg.state.Parameters) > 0 {
		validCount := 0
		for _, param := range tg.state.Parameters {
			if param.IsValid {
				validCount++
			}
		}
		if status.Len() > 0 {
			status.WriteString(" | ")
		}
		status.WriteString(fmt.Sprintf("Parameters: %d/%d valid", validCount, len(tg.state.Parameters)))
	}

	if tg.config.EnableKeyHelp {
		if status.Len() > 0 {
			status.WriteString(" | ")
		}
		status.WriteString(tg.keyBindings.GetHelpText(tg.config.Theme))
	}

	tg.statusBar.SetText(status.String())
}

func (tg *TemplateGenerator) updateReviewPane() {
	if tg.state.Template == nil {
		return
	}

	var review strings.Builder

	// Template information
	review.WriteString(fmt.Sprintf("[yellow]Template:[white] %s\n", tg.state.Template.Name))
	review.WriteString(fmt.Sprintf("[yellow]Description:[white] %s\n", tg.state.Template.Description))
	review.WriteString(fmt.Sprintf("[yellow]Category:[white] %s\n", tg.state.Template.Category))

	// Workspace context
	if tg.state.Context != nil {
		review.WriteString(fmt.Sprintf("\n[yellow]Workspace Context:[white]\n"))
		review.WriteString(fmt.Sprintf("  Directory: %s\n", tg.state.Context.WorkingDirectory))
		review.WriteString(fmt.Sprintf("  Language: %s\n", tg.state.Context.Language))
		if tg.state.Context.Framework != "" {
			review.WriteString(fmt.Sprintf("  Framework: %s\n", tg.state.Context.Framework))
		}
	}

	// Parameters
	if len(tg.state.Parameters) > 0 {
		review.WriteString(fmt.Sprintf("\n[yellow]Parameters:[white]\n"))
		for name, paramInput := range tg.state.Parameters {
			review.WriteString(fmt.Sprintf("  %s: %s\n", name, paramInput.Value))
		}
	}

	// Preview of template content (first few lines)
	review.WriteString(fmt.Sprintf("\n[yellow]Template Content (preview):[white]\n"))
	lines := strings.Split(tg.state.Template.Content, "\n")
	previewLines := 10
	if len(lines) > previewLines {
		lines = lines[:previewLines]
		lines = append(lines, "...")
	}
	for _, line := range lines {
		review.WriteString(fmt.Sprintf("  %s\n", line))
	}

	tg.reviewPane.SetText(review.String())
}

// Workspace context methods
func (tg *TemplateGenerator) initWorkspaceContext() {
	go func() {
		context, err := tg.manager.DetectWorkspaceContext()
		if err != nil {
			tg.logger.Warn("failed to detect workspace context", zap.Error(err))
			return
		}
		tg.state.Context = context
		tg.updateUI()
	}()
}

func (tg *TemplateGenerator) refreshWorkspaceContext() {
	tg.showStatus("Refreshing workspace context...", "info")
	tg.initWorkspaceContext()
}

// Utility methods
func (tg *TemplateGenerator) showStatus(message, msgType string) {
	statusMsg := NewStatusMessage(message, msgType, 3*time.Second)
	formattedMsg := statusMsg.FormatForDisplay(tg.config.Theme)
	
	// Temporarily update status bar
	originalText := tg.statusBar.GetText(false)
	tg.statusBar.SetText(formattedMsg)
	
	// Restore after duration
	go func() {
		time.Sleep(statusMsg.Duration)
		tg.statusBar.SetText(originalText)
	}()
}

func (tg *TemplateGenerator) showError(message string) {
	tg.showStatus(message, "error")
}

func (tg *TemplateGenerator) handleError(err error) {
	tg.logger.Error("template generator error", zap.Error(err))
	tg.showError(err.Error())
	
	if tg.callbacks.OnError != nil {
		tg.callbacks.OnError(err)
	}
}

// GetApplication returns the tview application
func (tg *TemplateGenerator) GetApplication() *tview.Application {
	// This would need to be set externally or passed in during initialization
	return nil
}

// Focus sets focus to the template generator
func (tg *TemplateGenerator) Focus(delegate func(p tview.Primitive)) {
	switch tg.state.Step {
	case StepSelectTemplate:
		if tg.templateBrowser != nil {
			tg.templateBrowser.Focus(delegate)
		}
	case StepCollectParameters:
		if tg.parameterForm != nil {
			delegate(tg.parameterForm)
		}
	case StepReviewAndGenerate:
		if tg.reviewPane != nil {
			delegate(tg.reviewPane)
		}
	case StepShowResult:
		if tg.resultPane != nil {
			delegate(tg.resultPane)
		}
	}
}

// HasFocus returns true if the template generator has focus
func (tg *TemplateGenerator) HasFocus() bool {
	switch tg.state.Step {
	case StepSelectTemplate:
		return tg.templateBrowser != nil && tg.templateBrowser.HasFocus()
	case StepCollectParameters:
		return tg.parameterForm != nil && tg.parameterForm.HasFocus()
	case StepReviewAndGenerate:
		return tg.reviewPane != nil && tg.reviewPane.HasFocus()
	case StepShowResult:
		return tg.resultPane != nil && tg.resultPane.HasFocus()
	}
	return false
}

// GetState returns the current generation state
func (tg *TemplateGenerator) GetState() *GenerationState {
	return tg.state
}

// SetTemplate sets the template and advances to parameter collection
func (tg *TemplateGenerator) SetTemplate(template prompt.Template) {
	tg.state.Template = &template
	tg.state.Step = StepCollectParameters
	tg.setupStepCollectParameters()
	tg.updateUI()
}

// Close cleans up the template generator
func (tg *TemplateGenerator) Close() {
	tg.cancel()
}