package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"openai-cli/internal/client"
	"openai-cli/internal/providers"
	"openai-cli/internal/service"
)

var (
	chatModel  string
	message    string
	promptMode bool
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Send chat completion request",
	RunE: func(cmd *cobra.Command, args []string) error {
		// interactive REPL mode
		if promptMode {
			// require password
			expected := os.Getenv("PROMPT_MODE_PASSWORD")
			if expected == "" {
				return fmt.Errorf("prompt mode password not set; define PROMPT_MODE_PASSWORD")
			}
			fmt.Print("Enter prompt-mode password: ")
			pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return err
			}
			if string(pwdBytes) != expected {
				return fmt.Errorf("invalid password")
			}
			reader := bufio.NewReader(os.Stdin)
			var conversation []client.ChatMessage
			endpoint := "/v1/chat/completions"
			for {
				fmt.Print(">>> ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				input = strings.TrimSpace(input)
				if input == "" {
					continue
				}
				if input == "exit" || input == "quit" {
					break
				}
				// append user message
				conversation = append(conversation, client.ChatMessage{Role: "user", Content: input})
				// prepare request payload
				reqPayload := client.ChatCompletionsRequest{Model: chatModel, Messages: conversation}
				var respMsg string
				// proxy via server URL
				if serverURL != "" {
					b, _ := json.Marshal(reqPayload)
					httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, serverURL+endpoint, bytes.NewReader(b))
					if err != nil {
						return err
					}
					httpReq.Header.Set("Content-Type", "application/json")
					httpReq.Header.Set("X-Provider", cfg.Provider)
					resp, err := http.DefaultClient.Do(httpReq)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						return fmt.Errorf("request failed: status code %d", resp.StatusCode)
					}
					var cResp client.ChatCompletionsResponse
					if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
						return err
					}
					respMsg = cResp.Choices[0].Message.Content
				} else {
					cliClient, err := providers.NewClientWithAuth(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.AgentURL)
					if err != nil {
						return err
					}
					svc := service.NewService(cliClient)
					cResp, err := svc.CreateChatCompletion(context.Background(), reqPayload)
					if err != nil {
						return err
					}
					respMsg = cResp.Choices[0].Message.Content
				}
				// output and append assistant message
				fmt.Println(respMsg)
				conversation = append(conversation, client.ChatMessage{Role: "assistant", Content: respMsg})
			}
			return nil
		}
		// build request payload
		reqPayload := client.ChatCompletionsRequest{
			Model:    chatModel,
			Messages: []client.ChatMessage{{Role: "user", Content: message}},
		}
		payloadBytes, err := json.Marshal(reqPayload)
		if err != nil {
			return err
		}
		endpoint := "/v1/chat/completions"
		// print curl command and exit
		if printCurl {
			data := strings.ReplaceAll(string(payloadBytes), "'", "\\'")
			var curlCmd string
			if serverURL != "" {
				curlCmd = fmt.Sprintf(
					"curl -X POST '%s%s' -H 'Content-Type: application/json' -H 'X-Provider: %s' -d '%s'",
					serverURL, endpoint, cfg.Provider, data)
			} else {
				curlCmd = fmt.Sprintf(
					"curl -X POST '%s%s' -H 'Content-Type: application/json' -H 'Authorization: Bearer %s' -d '%s'",
					cfg.BaseURL, endpoint, cfg.APIKey, data)
			}
			fmt.Println(curlCmd)
			return nil
		}
		// proxy via server URL
		if serverURL != "" {
			httpReq, err := http.NewRequestWithContext(
				context.Background(), http.MethodPost, serverURL+endpoint, bytes.NewReader(payloadBytes))
			if err != nil {
				return err
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("X-Provider", cfg.Provider)
			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("request failed: status code %d", resp.StatusCode)
			}
			var cResp client.ChatCompletionsResponse
			if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
				return err
			}
			fmt.Println(cResp.Choices[0].Message.Content)
			return nil
		}
		// direct provider client
		cliClient, err := providers.NewClientWithAuth(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.AgentURL)
		if err != nil {
			return err
		}
		svc := service.NewService(cliClient)
		resp, err := svc.CreateChatCompletion(context.Background(), reqPayload)
		if err != nil {
			return err
		}
		fmt.Println(resp.Choices[0].Message.Content)
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVar(&chatModel, "model", "gpt-3.5-turbo", "chat model to use")
	chatCmd.Flags().StringVar(&message, "message", "", "message content")
	chatCmd.Flags().BoolVar(&promptMode, "prompt-mode", false, "interactive prompt mode (REPL)")
	chatCmd.MarkFlagRequired("message")
}
