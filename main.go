package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"gotgbot/repository"
	database "gotgbot/repository/db"
	"gotgbot/usrbot"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gotgbot/bot"
	"gotgbot/config"
)

//go:embed repository/migrations/*.sql
var embedMigrations embed.FS

func main() {
	config, err := config.ReadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %s", err.Error())
	}

	db, err := database.ConnectAndMigrate(context.Background(), &config.Postgres, embedMigrations)
	if err != nil {
		log.Fatalf("db connect failed: %s", err.Error())
	}

	repo := repository.NewRepository(db, config)

	bot, err := bot.NewTgBot(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = bot.Start(rootCtx)
	if err != nil {
		log.Fatal(err.Error())
	}

	time.Sleep(time.Second)
	//bot.SendMessage(context.Background(), "привет \ndlfkjdk", 347)

	usr, err := usrbot.NewUsrBot(rootCtx, config, bot, repo)
	if err != nil {
		log.Fatalf("usrbot init failed: %s", err.Error())
	}

	err = usr.Start(rootCtx)
	if err != nil {
		if errors.Is(err, context.Canceled) && rootCtx.Err() == context.Canceled {
			fmt.Println("\rClosed")
			os.Exit(0)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
