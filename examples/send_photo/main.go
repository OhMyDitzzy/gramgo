package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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

	photoHandler(bot)

	// Start polling
	go func() {
		if err := bot.StartPolling(ctx); err != nil {
			panic(err)
		}
	}()
	log.Println("Bot is live!")

	<-ctx.Done()
	bot.Stop()
}

func photoHandler(bot *gramgo.GramGoBot) {
	bot.OnMessage(func(ctx *gramgo.Context) error {
		msg := ctx.Update.Message
		// Ignore command message
		if strings.HasPrefix(msg.Text, "/") {
			return nil
		}

		params := &types.SendPhotoParams{
			ChatID: msg.Chat.ID,
			Photo: &types.InputFileString{
				Data: "https://upload.wikimedia.org/wikipedia/commons/thumb/b/b9/2023_Facebook_icon.svg/200px-2023_Facebook_icon.svg.png",
			},
			Caption: "Facebook logo from URL",
		}
		_, err := bot.SendPhoto(ctx, params)
		return err
	})

	bot.OnCommand("yuki", func(ctx *gramgo.Context) error {
		msg := ctx.Update.Message
		// Read from disk
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Error getting current working directory: %v", err)
		}
		imgPath := "/examples/send_photo/media/yuki.png"
		filePath := fmt.Sprintf("%s%c%s", pwd, os.PathSeparator, imgPath)
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()
		params := &types.SendPhotoParams{
			ChatID: msg.Chat.ID,
			Photo: &types.InputFileUpload{
				Filename: "yuki.png",
				Data:     file,
			},
			Caption: "Yuki Souo",
		}

		_, err = bot.SendPhoto(ctx, params)
		return err
	})
}
