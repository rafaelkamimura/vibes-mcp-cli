package ui

import (
	"fmt"
	"time"

	"openai-cli/internal/terminal"

	"github.com/rivo/tview"
)

// SafeUIWrapper provides safe wrappers for tview components that handle TTY issues
type SafeUIWrapper struct {
	hasValidTTY bool
	initError   error
}

// NewSafeUIWrapper creates a new safe UI wrapper
func NewSafeUIWrapper() *SafeUIWrapper {
	wrapper := &SafeUIWrapper{
		hasValidTTY: terminal.HasTTY(),
	}
	
	// Pre-validate the environment
	if canRun, err := terminal.CanRunTUI(); !canRun {
		wrapper.initError = err
	}
	
	return wrapper
}

// IsValid returns true if the UI wrapper can safely create TUI components
func (w *SafeUIWrapper) IsValid() bool {
	return w.hasValidTTY && w.initError == nil
}

// GetError returns the initialization error if any
func (w *SafeUIWrapper) GetError() error {
	return w.initError
}

// SafeNewApplication creates a tview application with error handling
func (w *SafeUIWrapper) SafeNewApplication() (*tview.Application, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create application: %w", w.initError)
	}
	
	var app *tview.Application
	var err error
	
	// Use panic recovery for tview creation
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating application: %v", r)
			}
		}()
		
		app = tview.NewApplication()
	}()
	
	if err != nil {
		return nil, err
	}
	
	if app == nil {
		return nil, fmt.Errorf("failed to create tview application")
	}
	
	return app, nil
}

// SafeNewTextView creates a tview text view with error handling
func (w *SafeUIWrapper) SafeNewTextView() (*tview.TextView, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create text view: %w", w.initError)
	}
	
	var view *tview.TextView
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating text view: %v", r)
			}
		}()
		
		view = tview.NewTextView()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return view, nil
}

// SafeNewInputField creates a tview input field with error handling
func (w *SafeUIWrapper) SafeNewInputField() (*tview.InputField, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create input field: %w", w.initError)
	}
	
	var field *tview.InputField
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating input field: %v", r)
			}
		}()
		
		field = tview.NewInputField()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return field, nil
}

// SafeNewList creates a tview list with error handling
func (w *SafeUIWrapper) SafeNewList() (*tview.List, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create list: %w", w.initError)
	}
	
	var list *tview.List
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating list: %v", r)
			}
		}()
		
		list = tview.NewList()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return list, nil
}

// SafeNewTable creates a tview table with error handling
func (w *SafeUIWrapper) SafeNewTable() (*tview.Table, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create table: %w", w.initError)
	}
	
	var table *tview.Table
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating table: %v", r)
			}
		}()
		
		table = tview.NewTable()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return table, nil
}

// SafeNewFlex creates a tview flex with error handling
func (w *SafeUIWrapper) SafeNewFlex() (*tview.Flex, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create flex: %w", w.initError)
	}
	
	var flex *tview.Flex
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating flex: %v", r)
			}
		}()
		
		flex = tview.NewFlex()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return flex, nil
}

// SafeNewPages creates tview pages with error handling
func (w *SafeUIWrapper) SafeNewPages() (*tview.Pages, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create pages: %w", w.initError)
	}
	
	var pages *tview.Pages
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating pages: %v", r)
			}
		}()
		
		pages = tview.NewPages()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return pages, nil
}

// SafeNewTreeView creates a tview tree view with error handling
func (w *SafeUIWrapper) SafeNewTreeView() (*tview.TreeView, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create tree view: %w", w.initError)
	}
	
	var tree *tview.TreeView
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating tree view: %v", r)
			}
		}()
		
		tree = tview.NewTreeView()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return tree, nil
}

// SafeNewDropDown creates a tview dropdown with error handling
func (w *SafeUIWrapper) SafeNewDropDown() (*tview.DropDown, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create dropdown: %w", w.initError)
	}
	
	var dropdown *tview.DropDown
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating dropdown: %v", r)
			}
		}()
		
		dropdown = tview.NewDropDown()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return dropdown, nil
}

// SafeNewForm creates a tview form with error handling
func (w *SafeUIWrapper) SafeNewForm() (*tview.Form, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create form: %w", w.initError)
	}
	
	var form *tview.Form
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating form: %v", r)
			}
		}()
		
		form = tview.NewForm()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return form, nil
}

// SafeNewModal creates a tview modal with error handling
func (w *SafeUIWrapper) SafeNewModal() (*tview.Modal, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("cannot create modal: %w", w.initError)
	}
	
	var modal *tview.Modal
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic creating modal: %v", r)
			}
		}()
		
		modal = tview.NewModal()
	}()
	
	if err != nil {
		return nil, err
	}
	
	return modal, nil
}

// SafeRunApplication runs a tview application with comprehensive error handling
func (w *SafeUIWrapper) SafeRunApplication(app *tview.Application, root tview.Primitive) error {
	if !w.IsValid() {
		return fmt.Errorf("cannot run application: %w", w.initError)
	}
	
	if app == nil {
		return fmt.Errorf("application is nil")
	}
	
	if root == nil {
		return fmt.Errorf("root primitive is nil")
	}
	
	// Set up timeout for application startup
	startupDone := make(chan error, 1)
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				startupDone <- fmt.Errorf("application panic: %v", r)
			}
		}()
		
		// Try to run the application
		if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
			startupDone <- terminal.CreateTerminalError(err, "TUI execution failed")
		} else {
			startupDone <- nil
		}
	}()
	
	// Wait for startup with timeout
	select {
	case err := <-startupDone:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("application startup timed out - possible TTY access issue")
	}
}

// GetTerminalInfo returns terminal information for debugging
func (w *SafeUIWrapper) GetTerminalInfo() *terminal.TerminalInfo {
	return terminal.GetTerminalInfo()
}

// CanCreateTUI returns whether TUI components can be safely created
func (w *SafeUIWrapper) CanCreateTUI() bool {
	canRun, _ := terminal.CanRunTUI()
	return canRun && w.IsValid()
}

// ValidateBeforeCreate performs validation before creating any TUI components
func (w *SafeUIWrapper) ValidateBeforeCreate() error {
	if !w.hasValidTTY {
		return fmt.Errorf("no TTY available for TUI components")
	}
	
	if w.initError != nil {
		return fmt.Errorf("UI wrapper initialization failed: %w", w.initError)
	}
	
	return terminal.ValidateTerminalEnvironment()
}