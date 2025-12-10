package gramgo

import (
	"context"

	"github.com/OhMyDitzzy/gramgo/types"
)

// SendMessage https://core.telegram.org/bots/api#sendmessage
func (b *GramGoBot) SendMessage(ctx context.Context, params *types.SendMessageParams) (*types.Message, error) {
	msg := &types.Message{}
	err := b.rawRequest(ctx, "sendMessage", params, &msg)
	return msg, err
}

// SendPhoto https://core.telegram.org/bots/api#SendPhoto
func (b *GramGoBot) SendPhoto(ctx context.Context, params *types.SendPhotoParams) (*types.Message, error) {
	msg := &types.Message{}
	err := b.rawRequest(ctx, "sendPhoto", params, &msg)
	return msg, err
}

// https://core.telegram.org/bots/api#senddice
func (b *GramGoBot) SendDice(ctx context.Context, params *types.SendDiceParams) (*types.Message, error) {
	msg := &types.Message{}
	err := b.rawRequest(ctx, "sendDice", params, &msg)
	return msg, err
}
