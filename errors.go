package gramgo

import (
	"errors"
	"fmt"

	"github.com/OhMyDitzzy/gramgo/types"
)

var (
	// ErrEmptyToken is returned when bot token is empty
	ErrEmptyToken = errors.New("bot token cannot be empty")

	// ErrInvalidUpdateMode is returned when trying to start both polling and webhook
	ErrInvalidUpdateMode = errors.New("cannot use both polling and webhook simultaneously")

	// ErrNoHandler is returned when no handler is registered
	ErrNoHandler = errors.New("no handler registered")
)

// APIError represents an error from Telegram Bot API
type APIError struct {
	Code        int
	Description string
	Parameters  *types.ResponseParameters
}

func (e *APIError) Error() string {
	if e.Parameters != nil {
		if e.Parameters.RetryAfter > 0 {
			return fmt.Sprintf("telegram api error %d: %s (retry after %d seconds)",
				e.Code, e.Description, e.Parameters.RetryAfter)
		}
		if e.Parameters.MigrateToChatID != 0 {
			return fmt.Sprintf("telegram api error %d: %s (migrate to chat id: %d)",
				e.Code, e.Description, e.Parameters.MigrateToChatID)
		}
	}
	return fmt.Sprintf("telegram api error %d: %s", e.Code, e.Description)
}

// IsRetryableError checks if the error is retryable
func IsRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == 429 || apiErr.Code >= 500
	}
	return false
}

// GetRetryAfter returns retry after seconds from error
func GetRetryAfter(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Parameters != nil {
			return apiErr.Parameters.RetryAfter
		}
	}
	return 0
}