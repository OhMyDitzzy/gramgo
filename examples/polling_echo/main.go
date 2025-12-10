package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/OhMyDitzzy/gramgo"
	"github.com/OhMyDitzzy/gramgo/types"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot, err := gramgo.NewBot(gramgo.Config{
		Token: os.Getenv("EXAMPLE_BOT_TOKEN"),
	})

	if err != nil {
		// Do smth here
		panic(err)
	}

	echoHandlers(bot)

	// Start the polling
	go func() {
		fmt.Println("Starting polling...")
		if err := bot.StartPolling(ctx); err != nil {
			// You can do smth here.
			panic(err)
		}
	}()

	fmt.Println("Bot is running. Press Ctrl+C to stop.")

	// Shutdown gracefully
	<-ctx.Done()
	fmt.Println("Shutdown gracefully...")
	bot.Stop()
	fmt.Println("Bot succesfully stoped.")
}

func echoHandlers(bot *gramgo.GramGoBot) {
	bot.OnMessage(func(ctx *gramgo.Context) error {
		msg := ctx.Update.Message

		params := &types.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "You said: " + msg.Text,
		}

		_, err := bot.SendMessage(ctx, params)

		return err
	})
}

