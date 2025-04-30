package cmd

import (
   "bytes"
   "context"
   "encoding/json"
   "fmt"
   "io"
   "net/http"

   "github.com/gdamore/tcell/v2"
   "github.com/rivo/tview"
   "github.com/spf13/cobra"

   "openai-cli/internal/client"
   "openai-cli/internal/providers"
   "openai-cli/internal/service"
)

// uiChatModel is the chat model used in the UI
var uiChatModel string

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
}

// runUI initializes and runs the TUI
func runUI() error {
   app := tview.NewApplication()
   // maintain full conversation context
   var conversation []client.ChatMessage
   // declare variables for closure capture
   var input *tview.InputField
   var chatView *tview.TextView
   // initialize input
   input = tview.NewInputField().SetLabel("You: ").SetFieldWidth(0)
   // initialize chat view
   chatView = tview.NewTextView().SetScrollable(true)
   chatView.SetBorder(true).SetTitle("MCP Chat UI")
   // track scroll offset for navigation
   offset := 0
   // allow Esc to switch focus back to chatView
   input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
       if event.Key() == tcell.KeyEsc {
           app.SetFocus(chatView)
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
   // Layout
   flex := tview.NewFlex().SetDirection(tview.FlexRow).
       AddItem(chatView, 0, 1, false).
       AddItem(input, 1, 0, true)
   // Run
   if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
       return err
   }
   return nil
}