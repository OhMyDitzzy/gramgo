package gramgo

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/OhMyDitzzy/gramgo/types"
)

type PollingConfig struct {
	Timeout        int      // Polling timeout in seconds (default: 60)
	Limit          int      // Number of updates to fetch (1-100, default: 100)
	AllowedUpdates []string // List of update types to receive
	DropPending    bool     // Drop all pending updates on start
}

type GetUpdatesParams struct {
	Offset         int64    `json:"offset,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

// StartPolling starts the bot with long polling
func (b *GramGoBot) StartPolling(ctx context.Context, configs ...PollingConfig) error {
	if b.isRunning {
		return errors.New("bot is already running")
	}

	config := PollingConfig{
		Timeout: 60,
		Limit:   100,
	}

	if len(configs) > 0 {
		userConfig := configs[0]

		if userConfig.Timeout > 0 {
			config.Timeout = userConfig.Timeout
		}
		if userConfig.Limit > 0 {
			config.Limit = userConfig.Limit
		}
		if userConfig.AllowedUpdates != nil {
			config.AllowedUpdates = userConfig.AllowedUpdates
		}
		config.DropPending = userConfig.DropPending
	}

	if config.DropPending {
		if err := b.dropPendingUpdates(ctx); err != nil {
			log.Printf("Failed to drop pending updates: %v", err)
		}
	}

	b.isRunning = true
	b.stopChan = make(chan struct{})

	var offset int64 = 0

	for {
		select {
		case <-ctx.Done():
			b.isRunning = false
			return ctx.Err()
		case <-b.stopChan:
			b.isRunning = false
			return nil
		default:
			updates, err := b.getUpdates(ctx, GetUpdatesParams{
				Offset:         offset,
				Limit:          config.Limit,
				Timeout:        config.Timeout,
				AllowedUpdates: config.AllowedUpdates,
			})

			if err != nil {
				// Don't log if context was canceled (graceful shutdown)
				if ctx.Err() == nil {
					log.Printf("Failed to get updates: %v", err)
				}
				time.Sleep(3 * time.Second)
				continue
			}

			for _, update := range updates {
				offset = update.ID + 1

				go b.handleUpdate(ctx, &update)
			}
		}
	}
}

func (b *GramGoBot) getUpdates(ctx context.Context, params GetUpdatesParams) ([]types.Update, error) {
	var updates []types.Update
	err := b.rawRequest(ctx, "getUpdates", params, &updates)
	return updates, err
}

func (b *GramGoBot) dropPendingUpdates(ctx context.Context) error {
	_, err := b.getUpdates(ctx, GetUpdatesParams{
		Offset:  -1,
		Limit:   1,
		Timeout: 1,
	})
	return err
}

func (b *GramGoBot) handleUpdate(ctx context.Context, update *types.Update) {
	updateCtx := newContext(ctx, b, update)

	handler := HandlerFunc(b.routeUpdate)
	for i := len(b.middleware) - 1; i >= 0; i-- {
		handler = b.middleware[i](handler)
	}

	if err := handler(updateCtx); err != nil {
		log.Printf("Handler error: %v", err)
	}
}

func (b *GramGoBot) routeUpdate(ctx *Context) error {
	update := ctx.Update

	var handlerType string

	switch {
	case update.Message != nil:
		handlerType = "message"
	case update.EditedMessage != nil:
		handlerType = "edited_message"
	case update.ChannelPost != nil:
		handlerType = "channel_post"
	case update.CallbackQuery != nil:
		handlerType = "callback_query"
	case update.InlineQuery != nil:
		handlerType = "inline_query"
	default:
		handlerType = "other"
	}

	handlers, exists := b.handlers[handlerType]
	if !exists {
		return nil
	}

	for _, handler := range handlers {
		if err := handler.handle(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (b *GramGoBot) OnMessage(handler HandlerFunc, middleware ...MiddlewareFunc) {
	b.handlers["message"] = append(b.handlers["message"],
		newHandler(handler, middleware...))
}

func (b *GramGoBot) OnCallbackQuery(handler HandlerFunc, middleware ...MiddlewareFunc) {
	b.handlers["callback_query"] = append(b.handlers["callback_query"],
		newHandler(handler, middleware...))
}

func (b *GramGoBot) OnCommand(command string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	wrappedHandler := func(ctx *Context) error {
		if FilterCommand(command)(ctx.Update) {
			return handler(ctx)
		}
		return nil
	}

	b.OnMessage(wrappedHandler, middleware...)
}

func (b *GramGoBot) OnInlineQuery(handler HandlerFunc, middleware ...MiddlewareFunc) {
	b.handlers["inline_query"] = append(b.handlers["inline_query"],
		newHandler(handler, middleware...))
}

