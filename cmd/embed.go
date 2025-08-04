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
	embedModel string
	inputs     []string
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Get embeddings for input text",
	RunE: func(cmd *cobra.Command, args []string) error {
		// build request payload
		reqPayload := client.EmbeddingRequest{Model: embedModel, Input: inputs}
		payloadBytes, err := json.Marshal(reqPayload)
		if err != nil {
			return err
		}
		endpoint := "/v1/embeddings"
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
			var eResp client.EmbeddingResponse
			if err := json.NewDecoder(resp.Body).Decode(&eResp); err != nil {
				return err
			}
			for _, data := range eResp.Data {
				fmt.Printf("Index: %d, Embedding: %v\n", data.Index, data.Embedding)
			}
			return nil
		}
		// direct provider client
		cliClient, err := providers.NewClientWithAuth(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.AgentURL)
		if err != nil {
			return err
		}
		svc := service.NewService(cliClient)
		resp, err := svc.CreateEmbedding(context.Background(), reqPayload)
		if err != nil {
			return err
		}
		for _, data := range resp.Data {
			fmt.Printf("Index: %d, Embedding: %v\n", data.Index, data.Embedding)
		}
		return nil
	},
}

func init() {
	embedCmd.Flags().StringVar(&embedModel, "model", "text-embedding-ada-002", "embedding model to use")
	embedCmd.Flags().StringSliceVar(&inputs, "input", nil, "input texts for embeddings")
	embedCmd.MarkFlagRequired("input")
}
