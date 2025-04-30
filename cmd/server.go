package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openai-cli/internal/client"
	"openai-cli/internal/providers"
	"openai-cli/internal/service"
)

// serverCmd starts the MCP HTTP server, proxying requests to multiple providers.
// mcpHandler constructs the HTTP handler with all MCP endpoints.
func mcpHandler() http.Handler {
   mux := http.NewServeMux()
   // /v1/completions endpoint
   mux.HandleFunc("/v1/completions", func(w http.ResponseWriter, r *http.Request) {
       if r.Method != http.MethodPost {
           http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
           return
       }
       var req client.CompletionsRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           logger.Error("decode completion request failed", zap.Error(err))
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }
       provider := r.Header.Get("X-Provider")
       if provider == "" {
           provider = cfg.Provider
       }
       cliClient, err := providers.NewClient(provider, cfg.APIKey, cfg.BaseURL)
       if err != nil {
           logger.Error("invalid provider", zap.String("provider", provider), zap.Error(err))
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }
       svc := service.NewService(cliClient)
       resp, err := svc.CreateCompletion(r.Context(), req)
       if err != nil {
           logger.Error("completion error", zap.Error(err))
           http.Error(w, err.Error(), http.StatusInternalServerError)
           return
       }
       w.Header().Set("Content-Type", "application/json")
       json.NewEncoder(w).Encode(resp)
   })
   // /v1/chat/completions endpoint
   mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
       if r.Method != http.MethodPost {
           http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
           return
       }
       var req client.ChatCompletionsRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           logger.Error("decode chat request failed", zap.Error(err))
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }
       provider := r.Header.Get("X-Provider")
       if provider == "" {
           provider = cfg.Provider
       }
       cliClient, err := providers.NewClient(provider, cfg.APIKey, cfg.BaseURL)
       if err != nil {
           logger.Error("invalid provider", zap.String("provider", provider), zap.Error(err))
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }
       svc := service.NewService(cliClient)
       resp, err := svc.CreateChatCompletion(r.Context(), req)
       if err != nil {
           logger.Error("chat error", zap.Error(err))
           http.Error(w, err.Error(), http.StatusInternalServerError)
           return
       }
       w.Header().Set("Content-Type", "application/json")
       json.NewEncoder(w).Encode(resp)
   })
   // /v1/embeddings endpoint
   mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
       if r.Method != http.MethodPost {
           http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
           return
       }
       var req client.EmbeddingRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           logger.Error("decode embedding request failed", zap.Error(err))
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }
       provider := r.Header.Get("X-Provider")
       if provider == "" {
           provider = cfg.Provider
       }
       cliClient, err := providers.NewClient(provider, cfg.APIKey, cfg.BaseURL)
       if err != nil {
           logger.Error("invalid provider", zap.String("provider", provider), zap.Error(err))
           http.Error(w, err.Error(), http.StatusBadRequest)
           return
       }
       svc := service.NewService(cliClient)
       resp, err := svc.CreateEmbedding(r.Context(), req)
       if err != nil {
           logger.Error("embedding error", zap.Error(err))
           http.Error(w, err.Error(), http.StatusInternalServerError)
           return
       }
       w.Header().Set("Content-Type", "application/json")
       json.NewEncoder(w).Encode(resp)
   })
   return mux
}
// serverCmd starts the MCP HTTP server, proxying requests to multiple providers.
var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server (HTTP proxy for LLM providers)",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("%s:%d", serverHost, serverPort)
		mux := http.NewServeMux()

		// /v1/completions endpoint
		mux.HandleFunc("/v1/completions", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req client.CompletionsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				logger.Error("decode completion request failed", zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			provider := r.Header.Get("X-Provider")
			if provider == "" {
				provider = cfg.Provider
			}
			cliClient, err := providers.NewClient(provider, cfg.APIKey, cfg.BaseURL)
			if err != nil {
				logger.Error("invalid provider", zap.String("provider", provider), zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			svc := service.NewService(cliClient)
			resp, err := svc.CreateCompletion(r.Context(), req)
			if err != nil {
				logger.Error("completion error", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		// /v1/chat/completions endpoint
		mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req client.ChatCompletionsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				logger.Error("decode chat request failed", zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			provider := r.Header.Get("X-Provider")
			if provider == "" {
				provider = cfg.Provider
			}
			cliClient, err := providers.NewClient(provider, cfg.APIKey, cfg.BaseURL)
			if err != nil {
				logger.Error("invalid provider", zap.String("provider", provider), zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			svc := service.NewService(cliClient)
			resp, err := svc.CreateChatCompletion(r.Context(), req)
			if err != nil {
				logger.Error("chat error", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		// /v1/embeddings endpoint
		mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req client.EmbeddingRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				logger.Error("decode embedding request failed", zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			provider := r.Header.Get("X-Provider")
			if provider == "" {
				provider = cfg.Provider
			}
			cliClient, err := providers.NewClient(provider, cfg.APIKey, cfg.BaseURL)
			if err != nil {
				logger.Error("invalid provider", zap.String("provider", provider), zap.Error(err))
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			svc := service.NewService(cliClient)
			resp, err := svc.CreateEmbedding(r.Context(), req)
			if err != nil {
				logger.Error("embedding error", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		server := &http.Server{Addr: addr, Handler: mux}

		// graceful shutdown
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-stop
			logger.Info("shutting down server")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.Shutdown(ctx)
		}()

		logger.Info("starting MCP server", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		logger.Info("server stopped")
		return nil
	},
}

func init() {
	serverCmd.Flags().StringVar(&serverHost, "host", "127.0.0.1", "host for HTTP server")
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "port for HTTP server")
	rootCmd.AddCommand(serverCmd)
}
