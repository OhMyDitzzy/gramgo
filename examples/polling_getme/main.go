package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/OhMyDitzzy/gramgo"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot, err := gramgo.NewBot(gramgo.Config{
		Token: os.Getenv("EXAMPLE_BOT_TOKEN"),
	})

	if err != nil {
		// You might be able to do something here.
		log.Fatal(err)
	}

	user, _ := bot.GetMe(ctx)
	log.Printf("User: %#v\n", user)

	// Shutdown gracefully
	<-ctx.Done()
	log.Println("Shutting down gracefully...")
	bot.Stop()
	log.Println("Bot stopped successfully")
}
