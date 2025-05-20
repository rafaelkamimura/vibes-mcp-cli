package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"

	"openai-cli/internal/client"
	"openai-cli/internal/providers"
	"openai-cli/internal/service"
)

// uiChatModel is the chat model used in the UI
var (
	uiChatModel      string
	uiCollectionPath string
)

// uiCmd launches a terminal UI for interactive chat
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Interactive terminal UI for MCP chat",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Run the TUI application
		return runUI()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.Flags().StringVar(&uiChatModel, "model", "gpt-3.5-turbo", "chat model to use in UI")
	uiCmd.Flags().StringVar(&uiCollectionPath, "collection", "", "path to Postman collection JSON file to load")
}

// runUI initializes and runs the TUI
func runUI() error {
	app := tview.NewApplication()
	// maintain full conversation context
	var conversation []client.ChatMessage
	// declare variables for closure capture
	var input *tview.InputField
	var chatView *tview.TextView
	var dropdown *tview.DropDown
	var modelDropdown *tview.DropDown
	// initialize input
	input = tview.NewInputField().SetLabel("You: ").SetFieldWidth(0)
	// initialize chat view
	chatView = tview.NewTextView().SetScrollable(true)
	chatView.SetBorder(true).SetTitle("MCP Chat UI")
	templates := cfg.Templates
	if len(templates) == 0 {
		templates = []string{
			"Hey, whats up!",
			"Hows the weather in Brasilia - DF right now?",
		}
	}
	dropdown = tview.NewDropDown().
		SetLabel("Templates: ").
		SetOptions(templates, func(text string, index int) {
			input.SetText(text)
			app.SetFocus(input)
		})
	dropdown.SetBorder(true).SetTitle("Templates")
	dropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyTab:
			app.SetFocus(input)
			return nil
		}
		return event
	})
	modelOptions := []string{"o4-mini", "gpt-3.5-turbo", "codex-cli"}
	var modelIndex int
	for i, m := range modelOptions {
		if m == uiChatModel {
			modelIndex = i
			break
		}
	}
	modelDropdown = tview.NewDropDown().
		SetLabel("Model: ").
		SetOptions(modelOptions, func(text string, index int) {
			uiChatModel = text
			app.SetFocus(input)
		}).
		SetCurrentOption(modelIndex)
	modelDropdown.SetBorder(true).SetTitle("Models")
	modelDropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyTab:
			app.SetFocus(input)
			return nil
		}
		return event
	})
	// track scroll offset for navigation
	offset := 0
	// allow Esc to switch focus back to chatView
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			app.SetFocus(chatView)
			return nil
		case tcell.KeyTab:
			app.SetFocus(dropdown)
			return nil
		}
		return event
	})
	// enable vim-like scrolling in chatView
	chatView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j': // scroll down
				offset++
				chatView.ScrollTo(offset, 0)
				return nil
			case 'k': // scroll up
				if offset > 0 {
					offset--
				}
				chatView.ScrollTo(offset, 0)
				return nil
			case 'g': // go to top
				offset = 0
				chatView.ScrollToBeginning()
				return nil
			case 'G': // go to bottom
				chatView.ScrollToEnd()
				return nil
			}
		case tcell.KeyEsc: // back to input
			app.SetFocus(input)
			return nil
		case tcell.KeyTab:
			app.SetFocus(dropdown)
			return nil
		}
		return event
	})
	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		user := input.GetText()
		if user == "" {
			return
		}
		fmt.Fprintf(chatView, "You: %s\n", user)
		input.SetText("")
		// append user message to conversation
		conversation = append(conversation, client.ChatMessage{Role: "user", Content: user})
		// Prepare request with full conversation
		reqPayload := client.ChatCompletionsRequest{Model: uiChatModel, Messages: conversation}
		payloadBytes, _ := json.Marshal(reqPayload)
		endpoint := "/v1/chat/completions"
		// Dispatch request
		var respMsg string
		var err error
		if serverURL != "" {
			httpReq, e := http.NewRequestWithContext(context.Background(), http.MethodPost, serverURL+endpoint, bytes.NewReader(payloadBytes))
			if e != nil {
				err = e
			} else {
				httpReq.Header.Set("Content-Type", "application/json")
				httpReq.Header.Set("X-Provider", cfg.Provider)
				resp, e := http.DefaultClient.Do(httpReq)
				if e != nil {
					err = e
				} else {
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						body, _ := io.ReadAll(resp.Body)
						err = fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(body))
					} else {
						var cResp client.ChatCompletionsResponse
						e = json.NewDecoder(resp.Body).Decode(&cResp)
						if e != nil {
							err = e
						} else {
							respMsg = cResp.Choices[0].Message.Content
						}
					}
				}
			}
		} else {
			cliClient, e := providers.NewClient(cfg.Provider, cfg.APIKey, cfg.BaseURL)
			if e != nil {
				err = e
			} else {
				svc := service.NewService(cliClient)
				cResp, e := svc.CreateChatCompletion(context.Background(), reqPayload)
				if e != nil {
					err = e
				} else {
					respMsg = cResp.Choices[0].Message.Content
				}
			}
		}
		if err != nil {
			fmt.Fprintf(chatView, "Error: %v\n", err)
		} else {
			fmt.Fprintf(chatView, "Bot: %s\n", respMsg)
			// append assistant message to conversation
			conversation = append(conversation, client.ChatMessage{Role: "assistant", Content: respMsg})
		}
		chatView.ScrollToEnd()
	})
	chatFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(chatView, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(dropdown, 0, 1, false).
			AddItem(modelDropdown, 0, 1, false), 3, 0, false).
		AddItem(input, 1, 0, true)

	fileBrowser := tview.NewTreeView()
	fileBrowser.SetBorder(true)
	fileBrowser.SetTitle("Select Collection (.json)")

	var addNodes func(node *tview.TreeNode, path string)
	addNodes = func(node *tview.TreeNode, path string) {
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())
			child := tview.NewTreeNode(entry.Name()).
				SetReference(fullPath).
				SetSelectable(true)
			if entry.IsDir() {
				child.SetColor(tcell.ColorGreen)
				child.SetExpanded(false)
				addNodes(child, fullPath)
			}
			node.AddChild(child)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	rootNode := tview.NewTreeNode(cwd).
		SetReference(cwd).
		SetColor(tcell.ColorGreen).
		SetExpanded(true)
	addNodes(rootNode, cwd)
	fileBrowser.SetRoot(rootNode).SetCurrentNode(rootNode)

	postmanContent := tview.NewTextView()
	postmanContent.SetBorder(true)
	postmanContent.SetTitle("Collection")
	if uiCollectionPath != "" {
		postmanContent.Write([]byte(fmt.Sprintf("Loaded collection: %s\n", uiCollectionPath)))
	} else {
		postmanContent.Write([]byte("No collection loaded\n"))
	}

	fileBrowser.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference().(string)
		info, err := os.Stat(ref)
		if err != nil {
			return
		}
		if info.IsDir() {
			node.SetExpanded(!node.IsExpanded())
			return
		}
		if strings.HasSuffix(strings.ToLower(ref), ".json") {
			uiCollectionPath = ref
			postmanContent.Clear()
			fmt.Fprintf(postmanContent, "Loaded collection: %s\n", uiCollectionPath)
		}
	})

	postmanFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(fileBrowser, 0, 1, true).
		AddItem(postmanContent, 0, 2, false)

	chatTitle := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[::b]Chat[::-] (F1)")
	postmanTitle := tview.NewTextView().
		SetDynamicColors(true).
		SetText("Postman (F2)")
	modeBar := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(chatTitle, 0, 1, false).
		AddItem(postmanTitle, 0, 1, false)

	pages := tview.NewPages().
		AddPage("chat", chatFlex, true, true).
		AddPage("postman", postmanFlex, true, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(modeBar, 1, 0, false).
		AddItem(pages, 0, 1, true)

	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			pages.SwitchToPage("chat")
			chatTitle.SetText("[::b]Chat[::-] (F1)")
			postmanTitle.SetText("Postman (F2)")
			return nil
		case tcell.KeyF2:
			pages.SwitchToPage("postman")
			chatTitle.SetText("Chat (F1)")
			postmanTitle.SetText("[::b]Postman[::-] (F2)")
			return nil
		case tcell.KeyCtrlS:
			return nil
		}
		return event
	})

	if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
		return err
	}
	return nil
}
