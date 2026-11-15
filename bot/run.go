package bot

import (
	"context"
	"fmt"
	"gotgbot/model"
	"gotgbot/repository"
	"gotgbot/templates"
	"log"
	"strconv"
	"strings"
	"sync"

	"gotgbot/config"

	"github.com/gotd/td/tg"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

type (
	TgBot struct {
		config   *config.Config
		bot      *telego.Bot
		repo     *repository.Repository
		wg       *sync.WaitGroup
		handler  *th.BotHandler
		provider Provider
	}
	Provider interface {
		ListUserMsgs(ctx context.Context, userID, limit, offset int64) ([]*model.Message, error)
		FindUserByID(ctx context.Context, id int64) (*tg.User, error)
	}
)

const (
	MsgKeyListMessages = "key_list_messsages"
)

func NewTgBot(cfg *config.Config, repo *repository.Repository) (*TgBot, error) {
	bot, err := telego.NewBot(cfg.BotToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Errorf("failed to init new tg bot: %w", err)
	}

	return &TgBot{
		config: cfg,
		bot:    bot,
		repo:   repo,
		wg:     &sync.WaitGroup{},
	}, nil
}
func (b *TgBot) SetProvider(provider Provider) {
	b.provider = provider
}

func (b *TgBot) Start(ctx context.Context) error {
	updates, _ := b.bot.UpdatesViaLongPolling(ctx, nil)

	//bh.HandleMessage(handleAnyText, th.AnyMessageWithText())\
	b.handler = b.initHandlers(updates)
	if err := b.runHandler(ctx); err != nil {
		return fmt.Errorf("failed to run updater: %w", err)
	}

	return nil
}

func parsePageData(data string) (msgKey string, userID int64, page int) {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return
	}
	userID, err := strconv.ParseInt(parts[2], 0, 64)
	if err != nil {
		return
	}
	page, err = strconv.Atoi(parts[3])
	if err != nil {
		return
	}

	return parts[0], userID, page
}

func (b *TgBot) initHandlers(updates <-chan telego.Update) (bh *th.BotHandler) {
	bh, err := th.NewBotHandler(b.bot, updates)
	if err != nil {
		fmt.Errorf("failed to init new bot handler: %w", err)
	}

	bh.Handle(func(ctx *th.Context, update telego.Update) error {
		log.Printf("handle /chats")
		_, err = b.bot.SendMessage(ctx, b.makeChatsMessage(ctx, update.Message.Chat.ID))
		if err != nil {
			log.Printf("failed to send message: %s", err.Error())
		}
		return nil
	}, th.CommandEqual("chats"))

	// порядок важен: побеждает первый подошедший хендлер
	bh.HandleMessage(b.handleUsersShared, func(_ context.Context, u telego.Update) bool {
		return u.Message != nil && u.Message.UsersShared != nil
	})

	bh.HandleCallbackQuery(func(ctx *th.Context, q telego.CallbackQuery) error {
		return ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(q.ID).WithText("Недоступно"))
	}, th.CallbackDataPrefix("noop"))

	bh.HandleCallbackQuery(func(ctx *th.Context, q telego.CallbackQuery) error {
		switch m := q.Message.(type) {
		case *telego.Message:
			_, userID, page := parsePageData(q.Data)
			err := b.updateListMessagesMessage(ctx, m, userID, page)
			if err != nil {
				log.Printf("failed to update msg: %w", err)
			}
		}
		return ctx.Bot().AnswerCallbackQuery(ctx, tu.CallbackQuery(q.ID))
	}, th.CallbackDataPrefix("page"))

	return bh
}

func (b *TgBot) StartAndWait(ctx context.Context) error {
	if err := b.Start(ctx); err != nil {
		return err
	}
	b.wg.Wait()
	return nil
}

func (b *TgBot) runHandler(ctx context.Context) error {
	b.wg.Add(1)
	go func() {
		b.handler.Start()
		defer b.handler.Stop()
	}()
	return nil
}

func (b *TgBot) handleUsersShared(ctx *th.Context, message telego.Message) error {
	if len(message.UsersShared.Users) > 0 {
		u := message.UsersShared.Users[0]
		msg, err := b.makeListMessagesMessage(ctx, message.Chat.ID, u.UserID)
		if err != nil {
			log.Printf("failed to make list messages message: %s", err.Error())
			return nil
		}
		_, err = b.bot.SendMessage(ctx, msg)
		if err != nil {
			log.Printf("failed to send message: %s", err.Error())
		}
	}
	return nil
}

func (b *TgBot) updateListMessagesMessage(ctx *th.Context, msg *telego.Message, userID int64, page int) error {
	user, err := b.provider.FindUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	text, err := b.makeListMessagesContent(ctx, userID, user.Username, page)
	if err != nil {
		return fmt.Errorf("failed to make list messages content: %w", err)
	}

	_, err = ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:    msg.Chat.ChatID(),
		MessageID: msg.MessageID,
		Text:      text,
		ParseMode: telego.ModeHTML,
	})
	if err != nil {
		return fmt.Errorf("failed to update list messages content: %w", err)
	}
	_, err = ctx.Bot().EditMessageReplyMarkup(ctx, (&telego.EditMessageReplyMarkupParams{}).
		WithMessageID(msg.MessageID).
		WithChatID(msg.Chat.ChatID()).
		WithReplyMarkup(b.makeListMessagesKeyboard(ctx, userID, page)))
	if err != nil {
		return fmt.Errorf("failed to update list messages msg: %w", err)
	}
	return nil
}

func (b *TgBot) makeListMessagesContent(ctx *th.Context, userID int64, Username string, page int) (string, error) {
	offset := 0
	if page > 1 {
		offset = 20 * (page - 1)
	}

	msgs, err := b.provider.ListUserMsgs(ctx, userID, 20, int64(offset))
	if err != nil {
		return "", fmt.Errorf("failed to list user msgs: %w", err)
	}

	text, err := templates.ListMessagesTemplate(Username, msgs)
	if err != nil {
		return "", err
	}

	return text, nil
}

func (b *TgBot) makeListMessagesMessage(ctx *th.Context, toID int64, userID int64) (*telego.SendMessageParams, error) {
	keyboard := b.makeListMessagesKeyboard(ctx, userID, 1)

	user, err := b.provider.FindUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	text, err := b.makeListMessagesContent(ctx, userID, user.Username, 1)
	return tu.Message(
		tu.ID(toID),
		text,
	).WithReplyMarkup(keyboard).WithParseMode(telego.ModeHTML), nil
}

func (b *TgBot) makeListMessagesKeyboard(ctx context.Context, userID int64, page int) *telego.InlineKeyboardMarkup {
	var btn1, btn2 telego.InlineKeyboardButton

	if page == 1 {
		btn1 = tu.InlineKeyboardButton("назад").
			WithCallbackData("noop")
	} else {
		btn1 = tu.InlineKeyboardButton(fmt.Sprintf("назад(%d)", page-1)).
			WithCallbackData(fmt.Sprintf("page:%s:%d:%d", MsgKeyListMessages, userID, page-1))
	}

	btn2 = tu.InlineKeyboardButton(fmt.Sprintf("далее(%d)", page+1)).
		WithCallbackData(fmt.Sprintf("page:%s:%d:%d", MsgKeyListMessages, userID, page+1))

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			btn1,
			btn2,
		),
	)

	return keyboard
}

func (b *TgBot) makeChatsMessage(ctx context.Context, id int64) *telego.SendMessageParams {
	btn := tu.KeyboardButton("Выбрать личный чат").
		WithRequestUsers(&telego.KeyboardButtonRequestUsers{
			RequestID:       1,
			UserIsBot:       new(false),
			RequestUsername: new(true),
		})

	keyboard := tu.Keyboard(
		tu.KeyboardRow(
			btn,
		),
	).WithResizeKeyboard()

	message := tu.Message(
		tu.ID(id),
		"Выберите чат",
	).WithReplyMarkup(keyboard)

	return message
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
