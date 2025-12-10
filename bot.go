package gramgo

import (
	"context"
	"net/http"
	"time"

	"github.com/OhMyDitzzy/gramgo/types"
)

type GramGoBot struct {
	token      string
	apiURL     string
	middleware []MiddlewareFunc
	handlers   map[string][]handler
	client     *http.Client
	stopChan   chan struct{}
	isRunning  bool
}

type Config struct {
	Token      string
	APIBaseURL string        // default: https://api.telegram.org
	Timeout    time.Duration // default: 30s
	Client     *http.Client  // optional custom http client
}

// NewBot create a new bot instance
//
// Example:
//
//	   bot, err := gramgo.NewBot(gramgo.Config{ Token: "..." })
//
//		  if err != nil {
//		     // Do smth
//		  }
func NewBot(config Config) (*GramGoBot, error) {
	if config.Token == "" {
		return nil, ErrEmptyToken
	}

	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://api.telegram.org"
	}

	if config.Timeout == 0 {
		config.Timeout = 90 * time.Second
	}

	client := config.Client
	if client == nil {
		client = &http.Client{
			Timeout: config.Timeout,
		}
	}

	bot := &GramGoBot{
		token:     config.Token,
		apiURL:    config.APIBaseURL + "/bot" + config.Token,
		client:    client,
		handlers:  make(map[string][]handler),
		stopChan:  make(chan struct{}),
		isRunning: false,
	}

	return bot, nil
}

func (b *GramGoBot) GetMe(ctx context.Context) (*types.User, error) {
	user := &types.User{}
	err := b.rawRequest(ctx, "getMe", nil, user)
	return user, err
}

func (b *GramGoBot) Use(middleware ...MiddlewareFunc) {
	b.middleware = append(b.middleware, middleware...)
}

// Stop gracefully stops the bot
func (b *GramGoBot) Stop() {
	if !b.isRunning {
		return
	}

	close(b.stopChan)
	b.isRunning = false
}

// IsRunning returns whether the bot is currently running
func (b *GramGoBot) IsRunning() bool {
	return b.isRunning
}
