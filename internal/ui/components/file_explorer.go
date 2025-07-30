package components

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"openai-cli/internal/app/files"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FileExplorerMode represents different modes of operation
type FileExplorerMode int

const (
	ModeBrowse FileExplorerMode = iota // Normal browsing mode
	ModeSearch                         // Search mode
	ModeSelect                         // File selection mode
)

// FileExplorerAction represents actions that can be performed on files
type FileExplorerAction int

const (
	ActionView FileExplorerAction = iota // View file content
	ActionEdit                           // Edit file
	ActionSendToClaude                   // Send file to Claude Code
	ActionAddToSession                   // Add file to current session
	ActionCopy                           // Copy file path
	ActionDelete                         // Delete file
	ActionRename                         // Rename file
	ActionRefresh                        // Refresh directory
)

// FileExplorerConfig holds configuration for the file explorer
type FileExplorerConfig struct {
	RootPath        string // Root directory path
	AllowedPaths    []string
	ForbiddenPaths  []string
	ShowHiddenFiles bool
	MaxFileSize     int64
	EnableSearch    bool
	EnablePreview   bool
}

// FileExplorerCallbacks defines callback functions for file explorer events
type FileExplorerCallbacks struct {
	OnFileSelect    func(path string, fileType files.FileType)
	OnFileAction    func(action FileExplorerAction, path string)
	OnDirectoryChange func(path string)
	OnError         func(err error)
}

// FileExplorer provides a tview-based file explorer component
type FileExplorer struct {
	*tview.Flex

	// Configuration
	config    *FileExplorerConfig
	callbacks *FileExplorerCallbacks

	// Core components
	navigator *files.Navigator
	mode      FileExplorerMode

	// UI components
	treeView     *tview.TreeView
	previewView  *tview.TextView
	searchInput  *tview.InputField
	statusBar    *tview.TextView
	breadcrumb   *tview.TextView
	helpText     *tview.TextView

	// State
	currentNode    *files.FileNode
	searchResults  []*files.SearchResult
	selectedFiles  []string
	isSearchActive bool
	lastError      error

	// Context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewFileExplorer creates a new file explorer component
func NewFileExplorer(config *FileExplorerConfig, callbacks *FileExplorerCallbacks) *FileExplorer {
	if config == nil {
		cwd, _ := os.Getwd()
		config = &FileExplorerConfig{
			RootPath:        cwd,
			ShowHiddenFiles: false,
			MaxFileSize:     10 * 1024 * 1024, // 10MB
			EnableSearch:    true,
			EnablePreview:   true,
		}
	}

	if callbacks == nil {
		callbacks = &FileExplorerCallbacks{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	fe := &FileExplorer{
		Flex:      tview.NewFlex(),
		config:    config,
		callbacks: callbacks,
		mode:      ModeBrowse,
		ctx:       ctx,
		cancel:    cancel,
	}

	fe.initNavigator()
	fe.initUI()
	fe.setupKeyBindings()

	return fe
}

// initNavigator initializes the file navigator with security config
func (fe *FileExplorer) initNavigator() {
	securityConfig := &files.SecurityConfig{
		AllowedPaths:   fe.config.AllowedPaths,
		ForbiddenPaths: fe.config.ForbiddenPaths,
		MaxFileSize:    fe.config.MaxFileSize,
		AllowHidden:    fe.config.ShowHiddenFiles,
		MaxDepth:       20,
	}

	fe.navigator = files.NewNavigator(securityConfig)
	
	// Set root path
	if err := fe.navigator.SetRoot(fe.config.RootPath); err != nil {
		fe.handleError(fmt.Errorf("failed to set root path: %w", err))
	}
}

// initUI initializes the user interface components
func (fe *FileExplorer) initUI() {
	// Create tree view
	fe.treeView = tview.NewTreeView()
	fe.treeView.SetBorder(true)
	fe.treeView.SetTitle("📁 File Explorer")
	fe.treeView.SetTitleAlign(tview.AlignLeft)

	// Create preview view
	fe.previewView = tview.NewTextView()
	fe.previewView.SetBorder(true)
	fe.previewView.SetTitle("👁️ Preview")
	fe.previewView.SetTitleAlign(tview.AlignLeft)
	fe.previewView.SetScrollable(true)
	fe.previewView.SetWordWrap(true)

	// Create search input
	fe.searchInput = tview.NewInputField()
	fe.searchInput.SetLabel("🔍 Search: ")
	fe.searchInput.SetBorder(true)
	fe.searchInput.SetTitle("Search Files")

	// Create status bar
	fe.statusBar = tview.NewTextView()
	fe.statusBar.SetBorder(false)
	fe.statusBar.SetTextAlign(tview.AlignLeft)
	fe.statusBar.SetDynamicColors(true)

	// Create breadcrumb
	fe.breadcrumb = tview.NewTextView()
	fe.breadcrumb.SetBorder(false)
	fe.breadcrumb.SetTextAlign(tview.AlignLeft)
	fe.breadcrumb.SetDynamicColors(true)

	// Create help text
	fe.helpText = tview.NewTextView()
	fe.helpText.SetBorder(false)
	fe.helpText.SetTextAlign(tview.AlignCenter)
	fe.helpText.SetDynamicColors(true)
	fe.helpText.SetText("[yellow]Enter[white]=Open [yellow]M[white]=Claude [yellow]A[white]=Add [yellow]/[white]=Search [yellow]R[white]=Refresh [yellow]H[white]=Toggle Hidden")

	// Setup layout
	fe.setupLayout()
	
	// Initialize tree content
	fe.refreshTree()
	fe.updateBreadcrumb()
	fe.updateStatus()
}

// setupLayout arranges the UI components
func (fe *FileExplorer) setupLayout() {
	// Top section: breadcrumb
	topFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(fe.breadcrumb, 0, 1, false)

	// Middle section: tree view and preview
	middleFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	
	if fe.config.EnablePreview {
		middleFlex.AddItem(fe.treeView, 0, 1, true).
			AddItem(fe.previewView, 0, 1, false)
	} else {
		middleFlex.AddItem(fe.treeView, 0, 1, true)
	}

	// Bottom section: search, status, and help
	bottomFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fe.statusBar, 1, 0, false).
		AddItem(fe.helpText, 1, 0, false)

	// Main layout
	fe.SetDirection(tview.FlexRow).
		AddItem(topFlex, 1, 0, false).
		AddItem(middleFlex, 0, 1, true).
		AddItem(bottomFlex, 2, 0, false)

	// Initially hide search input
	fe.searchInput.SetBorder(false)
}

// setupKeyBindings configures keyboard shortcuts
func (fe *FileExplorer) setupKeyBindings() {
	// Tree view key bindings
	fe.treeView.SetSelectedFunc(fe.onNodeSelected)
	fe.treeView.SetInputCapture(fe.onTreeKeyPress)

	// Search input key bindings
	fe.searchInput.SetDoneFunc(fe.onSearchDone)
	fe.searchInput.SetInputCapture(fe.onSearchKeyPress)
	fe.searchInput.SetChangedFunc(fe.onSearchChanged)
}

// onTreeKeyPress handles key presses in the tree view
func (fe *FileExplorer) onTreeKeyPress(event *tcell.EventKey) *tcell.EventKey {
	node := fe.treeView.GetCurrentNode()
	if node == nil {
		return event
	}

	switch event.Key() {
	case tcell.KeyEnter:
		fe.onNodeSelected(node)
		return nil

	case tcell.KeyRune:
		switch event.Rune() {
		case '/':
			if fe.config.EnableSearch {
				fe.activateSearch()
			}
			return nil

		case 'h', 'H':
			fe.toggleHiddenFiles()
			return nil

		case 'r', 'R':
			fe.refreshCurrentDirectory()
			return nil

		case 'm', 'M':
			fe.sendToClaude(node)
			return nil

		case 'a', 'A':
			fe.addToSession(node)
			return nil

		case 'c', 'C':
			fe.copyPath(node)
			return nil

		case 'v', 'V':
			fe.previewFile(node)
			return nil

		case 'd', 'D':
			fe.deleteFile(node)
			return nil

		case 'n', 'N':
			fe.renameFile(node)
			return nil

		case 'q', 'Q':
			fe.quit()
			return nil
		}

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		fe.navigateUp()
		return nil

	case tcell.KeyCtrlR:
		fe.refreshTree()
		return nil
	}

	return event
}

// onSearchKeyPress handles key presses in the search input
func (fe *FileExplorer) onSearchKeyPress(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		fe.deactivateSearch()
		return nil
	}
	return event
}

// onSearchChanged handles search input changes (for real-time search)
func (fe *FileExplorer) onSearchChanged(text string) {
	if len(text) > 2 {
		go fe.performQuickSearch(text)
	}
}

// onSearchDone handles search input completion
func (fe *FileExplorer) onSearchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		pattern := fe.searchInput.GetText()
		if pattern != "" {
			go fe.performFullSearch(pattern)
		}
	}
}

// onNodeSelected handles node selection in the tree
func (fe *FileExplorer) onNodeSelected(node *tview.TreeNode) {
	if node == nil {
		return
	}

	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	fileInfo, err := fe.navigator.GetFileInfo(path)
	if err != nil {
		fe.handleError(err)
		return
	}

	fe.currentNode = fileInfo

	if fileInfo.IsDir {
		// Toggle directory expansion
		if err := fe.navigator.Navigate(path); err != nil {
			fe.handleError(err)
			return
		}

		node.SetExpanded(!node.IsExpanded())
		fe.loadDirectoryChildren(node, path)

		// Update navigation state
		fe.updateBreadcrumb()
		fe.updateStatus()

		if fe.callbacks.OnDirectoryChange != nil {
			fe.callbacks.OnDirectoryChange(path)
		}
	} else {
		// Handle file selection
		fe.previewFile(node)

		if fe.callbacks.OnFileSelect != nil {
			fe.callbacks.OnFileSelect(path, fileInfo.FileType)
		}
	}
}

// refreshTree rebuilds the entire tree structure
func (fe *FileExplorer) refreshTree() {
	root := fe.navigator.GetRoot()
	if root == nil {
		return
	}

	// Create root tree node
	rootNode := tview.NewTreeNode(root.GetDisplayName()).
		SetReference(root.Path).
		SetSelectable(true).
		SetExpanded(true).
		SetColor(tcell.ColorGreen)

	fe.treeView.SetRoot(rootNode)
	fe.loadDirectoryChildren(rootNode, root.Path)
}

// loadDirectoryChildren loads children for a directory node
func (fe *FileExplorer) loadDirectoryChildren(treeNode *tview.TreeNode, path string) {
	// Clear existing children
	treeNode.ClearChildren()

	// Get file node
	fileNode := fe.navigator.FindNode(path)
	if fileNode == nil {
		return
	}

	// Load children if needed
	if !fileNode.IsLoaded {
		if err := fe.navigator.LoadChildren(fileNode); err != nil {
			fe.handleError(err)
			return
		}
	}

	// Add child nodes to tree
	for _, child := range fileNode.Children {
		// Skip hidden files if not enabled
		if !fe.config.ShowHiddenFiles && strings.HasPrefix(child.Name, ".") {
			continue
		}

		childTreeNode := tview.NewTreeNode(child.GetIcon() + " " + child.GetDisplayName()).
			SetReference(child.Path).
			SetSelectable(true)

		if child.IsDir {
			childTreeNode.SetColor(tcell.ColorGreen)
			childTreeNode.SetExpanded(false)
		} else {
			// Set color based on file type
			childTreeNode.SetColor(fe.getFileTypeColor(child.FileType))
		}

		treeNode.AddChild(childTreeNode)
	}
}

// getFileTypeColor returns a color for the file type
func (fe *FileExplorer) getFileTypeColor(fileType files.FileType) tcell.Color {
	switch fileType {
	case files.FileTypeCode, files.FileTypeGo, files.FileTypePython, files.FileTypeJava:
		return tcell.ColorYellow
	case files.FileTypeMarkdown, files.FileTypeText:
		return tcell.ColorWhite
	case files.FileTypeJSON, files.FileTypeYAML, files.FileTypeXML:
		return tcell.ColorBlue
	case files.FileTypeImage:
		return tcell.ColorPurple
	case files.FileTypeArchive, files.FileTypeBinary:
		return tcell.ColorRed
	case files.FileTypeExecutable:
		return tcell.ColorGreen
	default:
		return tcell.ColorWhite
	}
}

// activateSearch enters search mode
func (fe *FileExplorer) activateSearch() {
	fe.mode = ModeSearch
	fe.isSearchActive = true

	// Replace status bar with search input
	fe.RemoveItem(fe.statusBar)
	fe.AddItem(fe.searchInput, 3, 0, false)

	// Focus on search input
	fe.GetApplication().SetFocus(fe.searchInput)

	fe.searchInput.SetBorder(true)
	fe.searchInput.SetTitle("🔍 Search Files")
}

// deactivateSearch exits search mode
func (fe *FileExplorer) deactivateSearch() {
	fe.mode = ModeBrowse
	fe.isSearchActive = false

	// Restore original layout
	fe.RemoveItem(fe.searchInput)
	fe.AddItem(fe.statusBar, 1, 0, false)

	// Clear search
	fe.searchInput.SetText("")
	fe.searchResults = nil

	// Focus back on tree
	fe.GetApplication().SetFocus(fe.treeView)

	fe.searchInput.SetBorder(false)
	fe.refreshTree()
}

// performQuickSearch performs a fast search for immediate feedback
func (fe *FileExplorer) performQuickSearch(pattern string) {
	results, err := fe.navigator.QuickSearch(fe.ctx, pattern)
	if err != nil {
		fe.handleError(err)
		return
	}

	fe.searchResults = results
	fe.displaySearchResults()
}

// performFullSearch performs a comprehensive search
func (fe *FileExplorer) performFullSearch(pattern string) {
	options := files.DefaultSearchOptions()
	options.Pattern = pattern
	options.MaxResults = 100

	results, err := fe.navigator.Search(fe.ctx, options)
	if err != nil {
		fe.handleError(err)
		return
	}

	fe.searchResults = results
	fe.displaySearchResults()
}

// displaySearchResults shows search results in the tree view
func (fe *FileExplorer) displaySearchResults() {
	if len(fe.searchResults) == 0 {
		return
	}

	// Create a virtual root for search results
	rootNode := tview.NewTreeNode("Search Results").
		SetSelectable(false).
		SetExpanded(true).
		SetColor(tcell.ColorYellow)

	for _, result := range fe.searchResults {
		icon := result.FileType.Icon()
		displayName := fmt.Sprintf("%s %s (%s)", icon, result.Name, result.MatchType.String())

		resultNode := tview.NewTreeNode(displayName).
			SetReference(result.Path).
			SetSelectable(true).
			SetColor(fe.getFileTypeColor(result.FileType))

		rootNode.AddChild(resultNode)
	}

	fe.treeView.SetRoot(rootNode)
}

// previewFile shows file preview in the preview pane
func (fe *FileExplorer) previewFile(node *tview.TreeNode) {
	if !fe.config.EnablePreview {
		return
	}

	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	
	// Clear preview
	fe.previewView.Clear()

	if !fe.navigator.IsTextFile(path) {
		fileType := fe.navigator.GetFileType(path)
		fe.previewView.SetText(fmt.Sprintf("📄 %s\n\nBinary file - cannot preview\nType: %s", 
			filepath.Base(path), fileType.String()))
		return
	}

	// Read file content
	content, err := fe.navigator.ReadFile(path)
	if err != nil {
		fe.previewView.SetText(fmt.Sprintf("❌ Error reading file:\n%v", err))
		return
	}

	// Set preview content
	fe.previewView.SetText(string(content))
	fe.previewView.SetTitle(fmt.Sprintf("👁️ Preview: %s", filepath.Base(path)))
}

// sendToClaude sends the selected file to Claude Code
func (fe *FileExplorer) sendToClaude(node *tview.TreeNode) {
	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	
	if fe.callbacks.OnFileAction != nil {
		fe.callbacks.OnFileAction(ActionSendToClaude, path)
	}

	fe.updateStatus("Sent to Claude Code: " + filepath.Base(path))
}

// addToSession adds the file to the current session
func (fe *FileExplorer) addToSession(node *tview.TreeNode) {
	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	fe.selectedFiles = append(fe.selectedFiles, path)

	if fe.callbacks.OnFileAction != nil {
		fe.callbacks.OnFileAction(ActionAddToSession, path)
	}

	fe.updateStatus("Added to session: " + filepath.Base(path))
}

// copyPath copies the file path to clipboard (simplified implementation)
func (fe *FileExplorer) copyPath(node *tview.TreeNode) {
	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	
	if fe.callbacks.OnFileAction != nil {
		fe.callbacks.OnFileAction(ActionCopy, path)
	}

	fe.updateStatus("Copied path: " + path)
}

// deleteFile deletes the selected file (with confirmation)
func (fe *FileExplorer) deleteFile(node *tview.TreeNode) {
	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	
	if fe.callbacks.OnFileAction != nil {
		fe.callbacks.OnFileAction(ActionDelete, path)
	}
}

// renameFile renames the selected file
func (fe *FileExplorer) renameFile(node *tview.TreeNode) {
	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	
	if fe.callbacks.OnFileAction != nil {
		fe.callbacks.OnFileAction(ActionRename, path)
	}
}

// toggleHiddenFiles toggles the display of hidden files
func (fe *FileExplorer) toggleHiddenFiles() {
	fe.config.ShowHiddenFiles = !fe.config.ShowHiddenFiles
	fe.refreshTree()
	
	status := "Hidden files: "
	if fe.config.ShowHiddenFiles {
		status += "shown"
	} else {
		status += "hidden"
	}
	fe.updateStatus(status)
}

// refreshCurrentDirectory refreshes the current directory
func (fe *FileExplorer) refreshCurrentDirectory() {
	currentPath := fe.navigator.GetCurrentPath()
	if currentPath == "" {
		return
	}

	// Find and refresh the current node
	if node := fe.navigator.FindNode(currentPath); node != nil {
		if err := fe.navigator.RefreshNode(node); err != nil {
			fe.handleError(err)
			return
		}
	}

	fe.refreshTree()
	fe.updateStatus("Directory refreshed")
}

// navigateUp navigates to the parent directory
func (fe *FileExplorer) navigateUp() {
	if err := fe.navigator.NavigateUp(); err != nil {
		fe.handleError(err)
		return
	}

	fe.refreshTree()
	fe.updateBreadcrumb()
	fe.updateStatus("Navigated up")
}

// updateBreadcrumb updates the breadcrumb display
func (fe *FileExplorer) updateBreadcrumb() {
	breadcrumb := fe.navigator.GetBreadcrumb()
	if len(breadcrumb) == 0 {
		fe.breadcrumb.SetText("")
		return
	}

	breadcrumbText := strings.Join(breadcrumb, " > ")
	fe.breadcrumb.SetText(fmt.Sprintf("📍 %s", breadcrumbText))
}

// updateStatus updates the status bar
func (fe *FileExplorer) updateStatus(message ...string) {
	currentPath := fe.navigator.GetCurrentPath()
	
	var statusText string
	if len(message) > 0 {
		statusText = message[0]
	} else {
		statusText = fmt.Sprintf("Path: %s", currentPath)
	}

	// Add navigation history info
	if fe.navigator.CanNavigateBack() || fe.navigator.CanNavigateForward() {
		historyInfo := ""
		if fe.navigator.CanNavigateBack() {
			historyInfo += "◀"
		}
		if fe.navigator.CanNavigateForward() {
			historyInfo += "▶"
		}
		statusText += fmt.Sprintf(" [%s]", historyInfo)
	}

	fe.statusBar.SetText(statusText)
}

// handleError handles errors by displaying them and optionally calling the error callback
func (fe *FileExplorer) handleError(err error) {
	fe.lastError = err
	fe.updateStatus(fmt.Sprintf("❌ Error: %v", err))

	if fe.callbacks.OnError != nil {
		fe.callbacks.OnError(err)
	}
}

// quit handles quit requests
func (fe *FileExplorer) quit() {
	fe.cancel()
}

// GetApplication returns the tview application if available
func (fe *FileExplorer) GetApplication() *tview.Application {
	// This is a helper method to get the application from the tree view
	// In a real implementation, you might want to store the application reference
	if fe.treeView != nil {
		// tview components don't directly expose the application
		// This would need to be set externally or passed in
		return nil
	}
	return nil
}

// Focus sets focus to the file explorer
func (fe *FileExplorer) Focus(delegate func(p tview.Primitive)) {
	delegate(fe.treeView)
}

// HasFocus returns true if the file explorer has focus
func (fe *FileExplorer) HasFocus() bool {
	return fe.treeView.HasFocus()
}

// SetCurrentPath sets the current path and refreshes the view
func (fe *FileExplorer) SetCurrentPath(path string) error {
	if err := fe.navigator.Navigate(path); err != nil {
		return err
	}

	fe.refreshTree()
	fe.updateBreadcrumb()
	fe.updateStatus()
	
	return nil
}

// GetCurrentPath returns the current path
func (fe *FileExplorer) GetCurrentPath() string {
	return fe.navigator.GetCurrentPath()
}

// GetSelectedFiles returns the list of selected files
func (fe *FileExplorer) GetSelectedFiles() []string {
	return fe.selectedFiles
}

// ClearSelectedFiles clears the list of selected files
func (fe *FileExplorer) ClearSelectedFiles() {
	fe.selectedFiles = nil
}