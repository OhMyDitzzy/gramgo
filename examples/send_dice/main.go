package main

import (
	"context"
	"log"
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
		panic(err)
	}

	// Register a handler
	diceHandler(bot)

	// start a polling
	go func() {
		if err := bot.StartPolling(ctx); err != nil {
			panic(err)
		}
	}()

	// gracefull stop
	<-ctx.Done()
	bot.Stop()
	log.Println("Stopping bot done.")
}

func diceHandler(bot *gramgo.GramGoBot) {
	bot.OnCommand("dice", func(ctx *gramgo.Context) error {
		msg := ctx.Update.Message
		_, err := bot.SendDice(ctx, &types.SendDiceParams{
			ChatID: msg.Chat.ID,
		})
		return err
	})
}
