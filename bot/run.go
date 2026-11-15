package bot

import (
	"context"
	"fmt"
	"log"
	"sync"

	"gotgbot/config"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type TgBot struct {
	config *config.Config
	bot    *telego.Bot
	wg     *sync.WaitGroup
}

func NewTgBot(cfg *config.Config) (*TgBot, error) {
	bot, err := telego.NewBot(cfg.BotToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Errorf("failed to init new tg bot: %w", err)
	}

	return &TgBot{
		config: cfg,
		bot:    bot,
		wg:     &sync.WaitGroup{},
	}, nil
}

func (b *TgBot) Start(ctx context.Context) error {
	if err := b.runUpdater(ctx); err != nil {
		return fmt.Errorf("failed to run updater: %w", err)
	}
	//b.wg.Wait()
	return nil
}
func (b *TgBot) StartAndWait(ctx context.Context) error {
	if err := b.Start(ctx); err != nil {
		return err
	}
	b.wg.Wait()
	return nil
}

func (b *TgBot) runUpdater(ctx context.Context) error {
	b.wg.Add(1)

	go func() {
		defer b.wg.Done()
		updates, _ := b.bot.UpdatesViaLongPolling(ctx, nil)

		// Loop through all updates when they came
		for update := range updates {
			log.Printf("tgbot update: %+v\n", update)
			if update.Message != nil {
				log.Printf("tgbot update message: %+v\n", update.Message)
				log.Printf("chatid: %d\n", update.Message.Chat.ChatID())
				log.Printf("userId: %d\n", update.Message.From.ID)
			}
		}
	}()
	return nil
}

func (b *TgBot) SendMessage(ctx context.Context, content string, chatid int64) error {
	chatid = 1393510348
	_, err := b.bot.SendMessage(ctx, tu.MessageWithEntities(
		tu.ID(1393510348),
		tu.Entity(content),
	))
	if err != nil {
		log.Printf("failed to send message: %s", err.Error())
		return err
	}

	return nil
}
