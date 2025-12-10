package gramgo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/OhMyDitzzy/gramgo/types"
)

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	URL                string   // HTTPS URL to send updates to (required for StartWebhook)
	ListenAddr         string   // Address to listen on (default: ":8443")
	Certificate        string   // Path to public key certificate
	IPAddress          string   // Fixed IP address
	MaxConnections     int      // Maximum allowed connections (1-100, default: 40)
	AllowedUpdates     []string // List of update types to receive
	DropPendingUpdates bool     // Drop all pending updates
	SecretToken        string   // Secret token for webhook validation
}

// SetWebhookParams represents parameters for setWebhook method
type SetWebhookParams struct {
	URL                string          `json:"url"`
	Certificate        types.InputFile `json:"certificate,omitempty"`
	IPAddress          string          `json:"ip_address,omitempty"`
	MaxConnections     int             `json:"max_connections,omitempty"`
	AllowedUpdates     []string        `json:"allowed_updates,omitempty"`
	DropPendingUpdates bool            `json:"drop_pending_updates,omitempty"`
	SecretToken        string          `json:"secret_token,omitempty"`
}

// WebhookInfo represents webhook info
type WebhookInfo = types.WebhookInfo

// SetWebhook sets a webhook
func (b *GramGoBot) SetWebhook(ctx context.Context, params *SetWebhookParams) error {
	var result bool
	return b.rawRequest(ctx, "setWebhook", params, &result)
}

// DeleteWebhook removes the webhook
func (b *GramGoBot) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error {
	params := struct {
		DropPendingUpdates bool `json:"drop_pending_updates,omitempty"`
	}{
		DropPendingUpdates: dropPendingUpdates,
	}

	var result bool
	return b.rawRequest(ctx, "deleteWebhook", params, &result)
}

// GetWebhookInfo gets current webhook status
func (b *GramGoBot) GetWebhookInfo(ctx context.Context) (*WebhookInfo, error) {
	info := &WebhookInfo{}
	err := b.rawRequest(ctx, "getWebhookInfo", nil, info)
	return info, err
}

// WebhookHandler returns an http.Handler for webhook
func (b *GramGoBot) WebhookHandler(secretToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secretToken != "" {
			token := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
			if token != secretToken {
				log.Printf("Invalid webhook secret token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read webhook body: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var update types.Update
		if err := json.Unmarshal(body, &update); err != nil {
			log.Printf("Failed to parse webhook update: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		go b.handleUpdate(r.Context(), &update)

		w.WriteHeader(http.StatusOK)
	})
}

// StartWebhook starts webhook server with optional config
func (b *GramGoBot) StartWebhook(ctx context.Context, configs ...WebhookConfig) error {
	if b.isRunning {
		return errors.New("bot is already running")
	}

	config := WebhookConfig{
		ListenAddr:     ":8443",
		MaxConnections: 40,
	}

	if len(configs) > 0 {
		userConfig := configs[0]
	
		if userConfig.URL != "" {
			config.URL = userConfig.URL
		}
		
		if userConfig.ListenAddr != "" {
			config.ListenAddr = userConfig.ListenAddr
		}
		if userConfig.Certificate != "" {
			config.Certificate = userConfig.Certificate
		}
		if userConfig.IPAddress != "" {
			config.IPAddress = userConfig.IPAddress
		}
		if userConfig.MaxConnections > 0 {
			config.MaxConnections = userConfig.MaxConnections
		}
		if userConfig.AllowedUpdates != nil {
			config.AllowedUpdates = userConfig.AllowedUpdates
		}
		config.DropPendingUpdates = userConfig.DropPendingUpdates
		config.SecretToken = userConfig.SecretToken
	}

	if config.URL == "" {
		return errors.New("webhook URL is required")
	}

	params := &SetWebhookParams{
		URL:                config.URL,
		IPAddress:          config.IPAddress,
		MaxConnections:     config.MaxConnections,
		AllowedUpdates:     config.AllowedUpdates,
		DropPendingUpdates: config.DropPendingUpdates,
		SecretToken:        config.SecretToken,
	}

	if config.Certificate != "" {
		// TODO: Load certificate file if provided
		// For now, user should set webhook manually if they need certificate
		
		log.Printf("Warning: Certificate loading not yet implemented")
	}

	if err := b.SetWebhook(ctx, params); err != nil {
		return err
	}

	b.isRunning = true
	b.stopChan = make(chan struct{})

	mux := http.NewServeMux()
	mux.Handle("/", b.WebhookHandler(config.SecretToken))

	server := &http.Server{
		Addr:         config.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		select {
		case <-ctx.Done():
		case <-b.stopChan:
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		b.isRunning = false
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		b.isRunning = false
		return err
	}

	return nil
}