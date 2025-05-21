package cmd

import (
   "bytes"
   "context"
   "encoding/json"
   "fmt"
   "github.com/spf13/cobra"
   "net/http"
   "openai-cli/internal/client"
   "openai-cli/internal/providers"
   "openai-cli/internal/service"
   "strings"
)

var (
	model  string
	prompt string
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate text completion",
	RunE: func(cmd *cobra.Command, args []string) error {
		// build request payload
		reqPayload := client.CompletionsRequest{Model: model, Prompt: prompt}
		payloadBytes, err := json.Marshal(reqPayload)
		if err != nil {
			return err
		}
		endpoint := "/v1/completions"
		// print curl command and exit
		if printCurl {
			data := strings.ReplaceAll(string(payloadBytes), "'", "\\'")
			var curlCmd string
			if serverURL != "" {
				curlCmd = fmt.Sprintf("curl -X POST '%s%s' -H 'Content-Type: application/json' -H 'X-Provider: %s' -d '%s'", serverURL, endpoint, cfg.Provider, data)
			} else {
				curlCmd = fmt.Sprintf("curl -X POST '%s%s' -H 'Content-Type: application/json' -H 'Authorization: Bearer %s' -d '%s'", cfg.BaseURL, endpoint, cfg.APIKey, data)
			}
			fmt.Println(curlCmd)
			return nil
		}
		// proxy via server URL if set
		if serverURL != "" {
			httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, serverURL+endpoint, bytes.NewReader(payloadBytes))
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
			var cResp client.CompletionsResponse
			if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
				return err
			}
			fmt.Println(cResp.Choices[0].Text)
			return nil
		}
		// direct provider client
		cliClient, err := providers.NewClient(cfg.Provider, cfg.APIKey, cfg.BaseURL)
		if err != nil {
			return err
		}
		svc := service.NewService(cliClient)
		resp, err := svc.CreateCompletion(context.Background(), reqPayload)
		if err != nil {
			return err
		}
		fmt.Println(resp.Choices[0].Text)
		return nil
	},
}

func init() {
	completionCmd.Flags().StringVar(&model, "model", "text-davinci-003", "model to use")
	completionCmd.Flags().StringVar(&prompt, "prompt", "", "prompt for completion")
	completionCmd.MarkFlagRequired("prompt")
}
