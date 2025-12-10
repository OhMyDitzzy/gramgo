package gramgo

import "github.com/OhMyDitzzy/gramgo/types"

type Update = types.Update

// UpdateFilter is a function that filters updates
type UpdateFilter func(*Update) bool

// FilterMessage returns true if update contains a message
func FilterMessage(update *Update) bool {
	return update.Message != nil
}

// FilterEditedMessage returns true if update contains an edited message
func FilterEditedMessage(update *Update) bool {
	return update.EditedMessage != nil
}

// FilterChannelPost returns true if update contains a channel post
func FilterChannelPost(update *Update) bool {
	return update.ChannelPost != nil
}

// FilterCallbackQuery returns true if update contains a callback query
func FilterCallbackQuery(update *Update) bool {
	return update.CallbackQuery != nil
}

// FilterInlineQuery returns true if update contains an inline query
func FilterInlineQuery(update *Update) bool {
	return update.InlineQuery != nil
}

// FilterCommand returns a filter that checks for specific command
func FilterCommand(command string) UpdateFilter {
	return func(update *Update) bool {
		if update.Message == nil || update.Message.Text == "" {
			return false
		}

		if len(update.Message.Entities) == 0 {
			return false
		}

		entity := update.Message.Entities[0]
		if entity.Type != "bot_command" || entity.Offset != 0 {
			return false
		}

		cmd := update.Message.Text[:entity.Length]
		return cmd == "/"+command || cmd == "/"+command+"@"
	}
}

// FilterText returns a filter that checks for specific text
func FilterText(text string) UpdateFilter {
	return func(update *Update) bool {
		if update.Message == nil {
			return false
		}
		return update.Message.Text == text
	}
}

// FilterChatType returns a filter that checks for specific chat type
func FilterChatType(chatType types.ChatType) UpdateFilter {
	return func(update *Update) bool {
		if update.Message == nil {
			return false
		}
		return update.Message.Chat.Type == chatType
	}
}

// FilterPrivateChat returns true if message is from private chat
func FilterPrivateChat(update *Update) bool {
	return FilterChatType(types.ChatTypePrivate)(update)
}

// FilterGroupChat returns true if message is from group chat
func FilterGroupChat(update *Update) bool {
	return FilterChatType(types.ChatTypeGroup)(update) ||
		FilterChatType(types.ChatTypeSupergroup)(update)
}

// FilterAny combines multiple filters with OR logic
func FilterAny(filters ...UpdateFilter) UpdateFilter {
	return func(update *Update) bool {
		for _, filter := range filters {
			if filter(update) {
				return true
			}
		}
		return false
	}
}

// FilterAll combines multiple filters with AND logic
func FilterAll(filters ...UpdateFilter) UpdateFilter {
	return func(update *Update) bool {
		for _, filter := range filters {
			if !filter(update) {
				return false
			}
		}
		return true
	}
}

// FilterNot inverts filter result
func FilterNot(filter UpdateFilter) UpdateFilter {
	return func(update *Update) bool {
		return !filter(update)
	}
}

