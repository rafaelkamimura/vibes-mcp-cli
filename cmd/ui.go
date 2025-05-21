package cmd

import (
   "bytes"
   "context"
   "encoding/json"
   "fmt"
   "io"
   "net/http"
   "net/url"
   "os"
   "path/filepath"
   "time"
   "unicode"

   "openai-cli/internal/client"
   "openai-cli/internal/mcp"
   "openai-cli/internal/providers"
   "openai-cli/internal/service"

   "github.com/gdamore/tcell/v2"
   "github.com/rivo/tview"
   "github.com/spf13/cobra"
)

// uiChatModel is the chat model used in the UI
var (
   uiChatModel     string
   uiExplorerRoot  string
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
	uiCmd.Flags().StringVar(&uiExplorerRoot, "explorer-root", "", "root path for file explorer")
}

// runUI initializes and runs the TUI
func runUI() error {
   // Ensure MCP server URL defaults to Agent backend if not explicitly set
   if serverURL == "" {
       serverURL = cfg.AgentURL
   }
   app := tview.NewApplication()
	pages := tview.NewPages()
	// maintain full conversation context
	var conversation []client.ChatMessage
	// declare variables for closure capture
	var input *tview.InputField
	var chatView *tview.TextView
	var dropdown *tview.DropDown
	var modelDropdown *tview.DropDown
   // login and registration forms share authentication state
   var loginForm *tview.Form
   var registerForm *tview.Form
   // file explorer components
   var explorerTree *tview.TreeView
   var fileContentView *tview.TextView
   var explorerFlex *tview.Flex
   var explorerHint *tview.TextView
   // track current page
   var currentPage string
	// rememberLogin controls whether a successful login should be saved to config
	var rememberLogin bool
	// homeList is the main menu list and updateHomeMenu rebuilds it on auth changes
	var homeList *tview.List
	var updateHomeMenu func()
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
	dropdown.SetBorder(true).
		SetTitle("Templates (Esc to cancel)").SetTitleAlign(tview.AlignLeft)
	dropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Map lower-case rune inputs to upper-case for case-insensitive option matching.
		if event.Key() == tcell.KeyRune {
			r := event.Rune()
			u := unicode.ToUpper(r)
			if u != r {
				event = tcell.NewEventKey(tcell.KeyRune, u, event.Modifiers())
			}
		}
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
	modelDropdown.SetBorder(true).
		SetTitle("Models (Esc to cancel)").SetTitleAlign(tview.AlignLeft)
	modelDropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Map lower-case rune inputs to upper-case for case-insensitive option matching.
		if event.Key() == tcell.KeyRune {
			r := event.Rune()
			u := unicode.ToUpper(r)
			if u != r {
				event = tcell.NewEventKey(tcell.KeyRune, u, event.Modifiers())
			}
		}
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
				if cfg.AuthToken != "" {
					httpReq.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
				}
				resp, e := http.DefaultClient.Do(httpReq)
				if e != nil {
					err = e
				} else {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusUnauthorized {
						modal := tview.NewModal().
							SetText("Session expired. Please login again.").
							AddButtons([]string{"OK"}).
							SetDoneFunc(func(_ int, _ string) {
								pages.RemovePage("authModal")
								cfg.AuthToken = ""
								pages.SwitchToPage("login")
								app.SetFocus(loginForm)
							})
						pages.AddPage("authModal", modal, true, true)
						return
					}
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

	agentView := tview.NewTextView().SetScrollable(true)
	agentView.SetBorder(true).SetTitle("Agent Chat UI")
	agentInput := tview.NewInputField().SetLabel("You: ").SetFieldWidth(0)
	agentInput.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		user := agentInput.GetText()
		if user == "" {
			return
		}
		fmt.Fprintf(agentView, "You: %s\n", user)
		agentInput.SetText("")
		body := map[string]string{"message": user}
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			fmt.Fprintf(agentView, "Error encoding request: %v\n", err)
		} else {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.AgentURL+"/agent/chat", buf)
			req.Header.Set("Content-Type", "application/json")
			if cfg.AuthToken != "" {
				req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Fprintf(agentView, "Error: %v\n", err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusUnauthorized {
					modal := tview.NewModal().
						SetText("Session expired. Please login again.").
						AddButtons([]string{"OK"}).
						SetDoneFunc(func(_ int, _ string) {
							pages.RemovePage("authModal")
							cfg.AuthToken = ""
							pages.SwitchToPage("login")
							app.SetFocus(loginForm)
						})
					pages.AddPage("authModal", modal, true, true)
				} else if resp.StatusCode != http.StatusOK {
					bodyBytes, _ := io.ReadAll(resp.Body)
					fmt.Fprintf(agentView, "Error: status %d: %s\n", resp.StatusCode, string(bodyBytes))
				} else {
					var agentResp struct {
						Response string `json:"response"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
						fmt.Fprintf(agentView, "Error decoding response: %v\n", err)
					} else {
						fmt.Fprintf(agentView, "Bot: %s\n", agentResp.Response)
					}
				}
			}
		}
		agentView.ScrollToEnd()
	})
	agentFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(agentView, 0, 1, false).
		AddItem(agentInput, 1, 0, true)

	// MCP view for JSON-RPC tool calls
	mcpView := tview.NewTextView().SetScrollable(true)
	mcpView.SetBorder(true).SetTitle("MCP Tool").SetTitleAlign(tview.AlignLeft)
	mcpInput := tview.NewInputField().SetLabel("Input: ").SetFieldWidth(0)
	mcpInput.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		inputText := mcpInput.GetText()
		if inputText == "" {
			return
		}
		fmt.Fprintf(mcpView, "You: %s\n", inputText)
		mcpInput.SetText("")
		traceID := fmt.Sprintf("%d", time.Now().UnixNano())
		// choose MCP server URL if provided, else use Agent backend
		var mcpClient *mcp.Client
		if serverURL != "" {
			mcpClient = mcp.NewClient(serverURL, cfg.AuthToken)
		} else {
			mcpClient = mcp.NewClient(cfg.AgentURL, cfg.AuthToken)
		}
		result, err := mcpClient.CallTool(context.Background(), inputText, traceID)
		if err != nil {
			fmt.Fprintf(mcpView, "Error: %v\n", err)
		} else {
			fmt.Fprintf(mcpView, "Result: %s\n", result)
		}
		mcpView.ScrollToEnd()
	})
	// Tools dropdown for selecting available MCP tools
	toolDropdown := tview.NewDropDown().
		SetLabel("Tool: ").
		SetOptions(cfg.Tools, func(text string, index int) {
			mcpInput.SetText(text + " ")
			app.SetFocus(mcpInput)
		}).
		SetCurrentOption(0)
	toolDropdown.SetBorder(true).
		SetTitle("Tools (Esc to cancel)").SetTitleAlign(tview.AlignLeft)
	toolDropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyTab {
			app.SetFocus(mcpInput)
			return nil
		}
		return event
	})
   // Layout for MCP view: results, tools dropdown, and input
   mcpFlex := tview.NewFlex().SetDirection(tview.FlexRow).
       AddItem(mcpView, 0, 1, false).
       AddItem(toolDropdown, 3, 0, false).
       AddItem(mcpInput, 1, 0, true)
   
   // File Explorer: browse and view files via MCP backend
   explorerTree = tview.NewTreeView()
   explorerTree.SetBorder(true).SetTitle("Explorer")
   // Recursively add directory nodes
   var addExplorerNodes func(node *tview.TreeNode, path string)
   addExplorerNodes = func(node *tview.TreeNode, path string) {
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
               child.SetColor(tcell.ColorGreen).SetExpanded(false)
               addExplorerNodes(child, fullPath)
           }
           node.AddChild(child)
       }
   }
   // Determine explorer root
   explorerRoot := uiExplorerRoot
   if explorerRoot == "" {
       cwd, err := os.Getwd()
       if err != nil {
           explorerRoot = "."
       } else {
           explorerRoot = cwd
       }
   }
   rootNode := tview.NewTreeNode(explorerRoot).
       SetReference(explorerRoot).
       SetColor(tcell.ColorGreen).
       SetExpanded(true)
   addExplorerNodes(rootNode, explorerRoot)
   explorerTree.SetRoot(rootNode).SetCurrentNode(rootNode)
   // File content view
   fileContentView = tview.NewTextView().SetScrollable(true)
   fileContentView.SetBorder(true).SetTitle("File Content")
   // On node selection: only toggle directories
   explorerTree.SetSelectedFunc(func(node *tview.TreeNode) {
       ref := node.GetReference().(string)
       info, err := os.Stat(ref)
       if err != nil {
           return
       }
       if info.IsDir() {
           node.SetExpanded(!node.IsExpanded())
       }
   })
   // Capture keys for viewing or invoking MCP on files
   explorerTree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
       node := explorerTree.GetCurrentNode()
       ref := node.GetReference().(string)
       info, err := os.Stat(ref)
       if err != nil {
           return event
       }
       // Enter to explore: toggle directories or load file locally
       if event.Key() == tcell.KeyEnter {
           if info.IsDir() {
               node.SetExpanded(!node.IsExpanded())
           } else {
               data, err := os.ReadFile(ref)
               fileContentView.Clear()
               if err != nil {
                   fmt.Fprintf(fileContentView, "Error reading file: %v\n", err)
               } else {
                   fmt.Fprintf(fileContentView, "%s", data)
               }
           }
           return nil
       }
       // 'm' key to switch to MCP mode for file
       if event.Key() == tcell.KeyRune && (event.Rune() == 'm' || event.Rune() == 'M') {
           if !info.IsDir() {
               // Compute relative path and prefill MCP input
               rel, err := filepath.Rel(explorerRoot, ref)
               if err != nil {
                   rel = ref
               }
               mcpInput.SetText(rel)
               currentPage = "mcp"
               pages.SwitchToPage("mcp")
               app.SetFocus(mcpInput)
           }
           return nil
       }
       return event
   })
   // Layout explorer flex
   explorerFlex = tview.NewFlex().SetDirection(tview.FlexColumn).
       AddItem(explorerTree, 0, 1, true).
       AddItem(fileContentView, 0, 2, false)

	menuTitle := tview.NewTextView()
	menuTitle.SetDynamicColors(true)
	menuTitle.SetText("[::b]Menu[::-] (F2)")
	menuTitle.SetTextAlign(tview.AlignCenter)
	homeHint := tview.NewTextView()
	homeHint.SetDynamicColors(true)
	homeHint.SetText("[::b]Home[::-] (F1)")
	homeHint.SetTextAlign(tview.AlignCenter)
   // Explorer hint for ModeBar
   explorerHint = tview.NewTextView()
   explorerHint.SetDynamicColors(true)
   explorerHint.SetText("[::b]Explorer[::-] (F3)")
   explorerHint.SetTextAlign(tview.AlignCenter)
   isAuthenticated := cfg.AuthToken != ""
   modeBar := tview.NewFlex().SetDirection(tview.FlexColumn)
   currentPage = "home"
	var updateModeBar func()
	updateModeBar = func() {
		modeBar.Clear().
			AddItem(homeHint, 0, 1, false)
		if isAuthenticated {
			modeBar.AddItem(menuTitle, 0, 1, false)
			modeBar.AddItem(explorerHint, 0, 1, false)
		}
	}
   updateModeBar()
  // Settings menu list and Tenants subpage
  settingsList := tview.NewList()
  tenantsList := tview.NewList()
  // Helper to load tenants
  loadTenants := func() {
      tenantsList.Clear()
      req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, cfg.AgentURL+"/user/tenants", nil)
      if err != nil {
          tenantsList.AddItem("Error: invalid request", "", 0, nil)
      } else {
          req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
          resp, err := http.DefaultClient.Do(req)
          if err != nil {
              tenantsList.AddItem("Error: fetch failed", "", 0, nil)
          } else {
              defer resp.Body.Close()
              if resp.StatusCode != http.StatusOK {
                  tenantsList.AddItem(fmt.Sprintf("Error: status %d", resp.StatusCode), "", 0, nil)
              } else {
                  var list []struct{ID, Name string}
                  if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
                      tenantsList.AddItem("Error: decode failed", "", 0, nil)
                  } else {
                      for _, t := range list {
                          tenantsList.AddItem(fmt.Sprintf("%s (%s)", t.Name, t.ID), "", 0, nil)
                      }
                  }
              }
          }
      }
      tenantsList.AddItem("Back", "Return to Settings", 'B', func() {
          currentPage = "settings"
          pages.SwitchToPage("settings")
          app.SetFocus(settingsList)
          updateModeBar()
      })
  }
  settingsList.
      AddItem("View Tenants", "List all tenants", 'T', func() {
          loadTenants()
          currentPage = "settingsTenants"
          pages.SwitchToPage("settingsTenants")
          app.SetFocus(tenantsList)
          updateModeBar()
      }).
      AddItem("Back", "Return to Home", 'B', func() {
          currentPage = "home"
          pages.SwitchToPage("home")
          app.SetFocus(homeList)
          updateModeBar()
      })
  settingsList.SetBorder(true).SetTitle("Settings").SetTitleAlign(tview.AlignCenter)
  tenantsList.SetBorder(true).SetTitle("Tenants").SetTitleAlign(tview.AlignLeft)

   // Dropdown menu for page navigation
   menuList := tview.NewList().
		AddItem("Chat", "Start interactive chat", 'C', func() {
			currentPage = "chat"
			pages.SwitchToPage("chat")
			app.SetFocus(input)
			updateModeBar()
			pages.RemovePage("menu")
		}).
       // Explorer
       AddItem("Explorer", "Browse files", 'E', func() {
           currentPage = "explorer"
           pages.SwitchToPage("explorer")
           app.SetFocus(explorerTree)
           updateModeBar()
           pages.RemovePage("menu")
       }).
       AddItem("Agent", "Start agent interactive chat", 'A', func() {
           currentPage = "agent"
           pages.SwitchToPage("agent")
           app.SetFocus(agentInput)
           updateModeBar()
           pages.RemovePage("menu")
       }).
       AddItem("MCP", "Invoke JSON-RPC tool", 'M', func() {
           currentPage = "mcp"
           pages.SwitchToPage("mcp")
           app.SetFocus(mcpInput)
           updateModeBar()
           pages.RemovePage("menu")
       }).
       AddItem("Settings", "Admin settings", 'S', func() {
           pages.SwitchToPage("settings")
           app.SetFocus(settingsList)
           updateModeBar()
           pages.RemovePage("menu")
       }).
       AddItem("Cancel", "", 0, func() {
			pages.RemovePage("menu")
			switch currentPage {
			case "chat":
				app.SetFocus(input)
       case "explorer":
           app.SetFocus(explorerTree)
       		case "agent":
       			app.SetFocus(agentInput)
       		case "mcp":
       			app.SetFocus(mcpInput)
			default:
				app.SetFocus(homeList)
			}
			updateModeBar()
		})
	menuList.SetBorder(true).
		SetTitle("Menu (Esc to cancel)").SetTitleAlign(tview.AlignCenter).
		SetBorderPadding(1, 1, 2, 2)
	menuList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Map lower-case rune inputs to upper-case for case-insensitive item matching.
		if event.Key() == tcell.KeyRune {
			r := event.Rune()
			u := unicode.ToUpper(r)
			if u != r {
				event = tcell.NewEventKey(tcell.KeyRune, u, event.Modifiers())
			}
		}
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("menu")
			switch currentPage {
			case "chat":
				app.SetFocus(input)
			case "explorer":
				app.SetFocus(explorerTree)
			case "agent":
				app.SetFocus(agentInput)
			case "mcp":
				app.SetFocus(mcpInput)
			default:
				app.SetFocus(homeList)
			}
			updateModeBar()
			return nil
		}
		return event
	})

	// Login form for authentication
	loginForm = tview.NewForm()
	loginForm.AddInputField("Username", "", 20, nil, nil)
	loginForm.AddPasswordField("Password", "", 20, '*', nil)
	loginForm.AddCheckbox("Remember me", false, func(checked bool) {
		rememberLogin = checked
	})
	loginForm.AddButton("Login", func() {
		username := loginForm.GetFormItemByLabel("Username").(*tview.InputField).GetText()
		password := loginForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
		form := url.Values{}
		form.Set("username", username)
		form.Set("password", password)
		form.Set("scope", "agent:chat")
		resp, err := http.PostForm(cfg.AgentURL+"/auth/login", form)
		msg := ""
		if err != nil {
			msg = fmt.Sprintf("Error: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				msg = fmt.Sprintf("Login failed (%d): %s", resp.StatusCode, string(body))
			} else {
				var tokenResp client.Token
				if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
					msg = fmt.Sprintf("Parse error: %v", err)
				} else {
					cfg.AuthToken = tokenResp.AccessToken
					if rememberLogin {
						if err := cfg.Save(); err != nil {
							msg = fmt.Sprintf("Login successful, but failed to save token: %v", err)
						} else {
							msg = "Login successful (token saved)"
						}
					} else {
						msg = "Login successful"
					}
				}
			}
		}
		modal := tview.NewModal().
			SetText(msg).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				pages.RemovePage("loginModal")
				if cfg.AuthToken != "" {
					isAuthenticated = true
					updateHomeMenu()
					updateModeBar()
					pages.SwitchToPage("home")
					app.SetFocus(homeList)
				} else {
					pages.SwitchToPage("login")
					app.SetFocus(loginForm)
				}
			})
		pages.AddPage("loginModal", modal, true, true)
	})
	loginForm.AddButton("Cancel", func() { app.Stop() })
	loginForm.SetBorder(true).SetTitle("Login").SetTitleAlign(tview.AlignLeft)

	// Registration form for new users
	registerForm = tview.NewForm()
	registerForm.AddInputField("Username", "", 20, nil, nil)
	registerForm.AddPasswordField("Password", "", 20, '*', nil)
	registerForm.AddInputField("Full Name", "", 30, nil, nil)
	registerForm.AddButton("Register", func() {
		username := registerForm.GetFormItemByLabel("Username").(*tview.InputField).GetText()
		password := registerForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
		fullName := registerForm.GetFormItemByLabel("Full Name").(*tview.InputField).GetText()
		body := client.UserCreate{Username: username, Password: password, FullName: fullName}
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			msg := fmt.Sprintf("Encode error: %v", err)
			modal := tview.NewModal().
				SetText(msg).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(_ int, _ string) {
					pages.RemovePage("registerModal")
					pages.SwitchToPage("register")
					app.SetFocus(registerForm)
				})
			pages.AddPage("registerModal", modal, true, true)
			return
		}
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, cfg.AgentURL+"/auth/register", buf)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		msg := ""
		if err != nil {
			msg = fmt.Sprintf("Error: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				bodyBytes, _ := io.ReadAll(resp.Body)
				msg = fmt.Sprintf("Register failed (%d): %s", resp.StatusCode, string(bodyBytes))
			} else {
				msg = "Registration successful"
			}
		}
		modal := tview.NewModal().
			SetText(msg).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				pages.RemovePage("registerModal")
				if resp != nil && resp.StatusCode == http.StatusCreated {
					pages.SwitchToPage("login")
					app.SetFocus(loginForm)
				} else {
					pages.SwitchToPage("register")
					app.SetFocus(registerForm)
				}
			})
		pages.AddPage("registerModal", modal, true, true)
	})
	registerForm.AddButton("Cancel", func() { app.Stop() })
	registerForm.SetBorder(true).SetTitle("Register").SetTitleAlign(tview.AlignLeft)

	homeList = tview.NewList()
	homeList.SetBorder(true).
		SetTitle("Home").
		SetTitleAlign(tview.AlignCenter).
		SetBorderPadding(1, 1, 2, 2)

	// Helper to (re)build the home menu based on authentication state
	updateHomeMenu = func() {
		homeList.Clear()
		if isAuthenticated {
			homeList.AddItem("Chat", "Start interactive chat", 'C', func() {
				currentPage = "chat"
				pages.SwitchToPage("chat")
				app.SetFocus(input)
				updateModeBar()
			})
			homeList.AddItem("Agent", "Start agent interactive chat", 'A', func() {
				currentPage = "agent"
				pages.SwitchToPage("agent")
				app.SetFocus(agentInput)
				updateModeBar()
			})
				homeList.AddItem("Explorer", "Browse files", 'E', func() {
				currentPage = "explorer"
				pages.SwitchToPage("explorer")
				app.SetFocus(explorerTree)
				updateModeBar()
			})
			homeList.AddItem("Logout", "Logout current session", 'O', func() {
				isAuthenticated = false
				cfg.AuthToken = ""
				updateModeBar()
				updateHomeMenu()
				pages.SwitchToPage("login")
				app.SetFocus(loginForm)
			})
		} else {
			homeList.AddItem("Login", "Authenticate with Agent", 'L', func() {
				pages.SwitchToPage("login")
				app.SetFocus(loginForm)
			})
			homeList.AddItem("Register", "Create new account", 'R', func() {
				pages.SwitchToPage("register")
				app.SetFocus(registerForm)
			})
		}
		homeList.AddItem("Quit", "Exit application", 'Q', func() {
			app.Stop()
		})
	}
	updateHomeMenu()
	// Convert lower-case rune inputs to upper-case to make menu shortcuts case-insensitive.
	homeList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			r := event.Rune()
			u := unicode.ToUpper(r)
			if u != r {
				return tcell.NewEventKey(tcell.KeyRune, u, event.Modifiers())
			}
		}
		return event
	})

	pages = pages.
		AddPage("home", homeList, true, true).
		AddPage("login", loginForm, true, false).
		AddPage("register", registerForm, true, false).
		AddPage("chat", chatFlex, true, false).
		AddPage("agent", agentFlex, true, false).
		AddPage("mcp", mcpFlex, true, false).
		AddPage("settings", settingsList, true, false).
		AddPage("settingsTenants", tenantsList, true, false).
		AddPage("explorer", explorerFlex, true, false)

	// Initial authentication: require login on startup
	if !isAuthenticated {
		pages.SwitchToPage("login")
		app.SetFocus(loginForm)
	}

	// header banner
	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetTextAlign(tview.AlignCenter)
	header.SetBorder(true)
	header.SetTitle(" Vibes CLI UI Home ")
	header.SetTitleAlign(tview.AlignCenter)
	header.SetText("🌊 [::b]Vibes CLI UI Home[::-] 🌊")
	header.SetBorderPadding(1, 1, 2, 2)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 5, 0, false).
		AddItem(modeBar, 1, 0, false).
		AddItem(pages, 0, 1, true)

	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			currentPage = "home"
			pages.SwitchToPage("home")
			app.SetFocus(homeList)
			updateModeBar()
			return nil
		case tcell.KeyF2:
			if isAuthenticated {
				pages.RemovePage("menu")
				pages.AddPage("menu", menuList, true, true)
				app.SetFocus(menuList)
			}
			return nil
       case tcell.KeyF3:
           if isAuthenticated {
               currentPage = "explorer"
               pages.SwitchToPage("explorer")
               app.SetFocus(explorerTree)
               updateModeBar()
           }
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
