package gramgo

import (
	"context"
	"log"
	"time"
)

type Context struct {
	context.Context
	Bot    *GramGoBot
	Update *Update
	Data   map[string]any
}

func newContext(ctx context.Context, bot *GramGoBot, update *Update) *Context {
	return &Context{
		Context: ctx,
		Bot:     bot,
		Update:  update,
		Data:    make(map[string]any),
	}
}

func (c *Context) set(key string, value any) {
	c.Data[key] = value
}

func (c *Context) get(key string) (any, bool) {
	val, ok := c.Data[key]
	return val, ok
}

type HandlerFunc func(*Context) error

type MiddlewareFunc func(HandlerFunc) HandlerFunc

type handler struct {
	handler    HandlerFunc
	middleware []MiddlewareFunc
}

func newHandler(handlr HandlerFunc, middleware ...MiddlewareFunc) handler {
	return handler{
		handler:    handlr,
		middleware: middleware,
	}
}

func (h handler) handle(ctx *Context) error {
	final := h.handler

	for i := len(h.middleware) - 1; i >= 0; i-- {
		final = h.middleware[i](final)
	}

	return final(ctx)
}

func Logger() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) error {
			start := time.Now()
			log.Printf("[gramgo] Processing update #%d", ctx.Update.ID)

			err := next(ctx)

			duration := time.Since(start)
			if err != nil {
				log.Printf("[gramgo] Update #%d failed in %v: %v", ctx.Update.ID, duration, err)
			} else {
				log.Printf("[gramgo] Update #%d completed in %v", ctx.Update.ID, duration)
			}

			return err
		}
	}
}

func Recovery() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[gramgo] Panic recovered: %v", r)
					err = nil
				}
			}()
			return next(ctx)
		}
	}
}

func Timeout(timeout time.Duration) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) error {
			timeoutCtx, cancel := context.WithTimeout(ctx.Context, timeout)
			defer cancel()

			ctx.Context = timeoutCtx

			errChan := make(chan error, 1)
			go func() {
				errChan <- next(ctx)
			}()

			select {
			case err := <-errChan:
				return err
			case <-timeoutCtx.Done():
				log.Printf("[gramgo] Handler timeout after %v", timeout)
				return timeoutCtx.Err()
			}
		}
	}
}

func RateLimit(maxRequests int, duration time.Duration) MiddlewareFunc {
	type userLimit struct {
		count     int
		resetTime time.Time
	}

	limits := make(map[int64]*userLimit)

	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) error {
			var userID int64

			if ctx.Update.Message != nil && ctx.Update.Message.From != nil {
				userID = ctx.Update.Message.From.ID
			} else if ctx.Update.CallbackQuery != nil {
				userID = ctx.Update.CallbackQuery.From.ID
			}

			if userID == 0 {
				return next(ctx)
			}

			now := time.Now()
			limit, exists := limits[userID]

			if !exists || now.After(limit.resetTime) {
				limits[userID] = &userLimit{
					count:     1,
					resetTime: now.Add(duration),
				}
				return next(ctx)
			}

			if limit.count >= maxRequests {
				log.Printf("[gramgo] Rate limit exceeded for user %d", userID)
				return nil
			}

			limit.count++
			return next(ctx)
		}
	}
}
