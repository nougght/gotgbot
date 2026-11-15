package usrbot

import (
	"context"
	"errors"
	"fmt"
	"gotgbot/model"
	"gotgbot/repository"
	"gotgbot/templates"
	"log"
	"maps"
	"slices"
	"time"

	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"
)

func (b *UsrBot) NewMessagesHandler(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
	go func() {
		b.savingNewMsgs.Store(true)
		defer b.savingNewMsgs.Store(false)
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return
		}

		// Use PeerID to find peer because *Short updates does not contain any entities, so it necessary to
		// store some entities.
		//
		// Storage can be filled using PeerCollector (i.e. fetching all dialogs first).
		p, err := storage.FindPeer(ctx, b.peerDB, msg.GetPeerID())
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Printf("%#+v: %s\n", p, msg.Message)

		saved, err := b.repo.MessagesRepo.AddMessage(ctx, model.NewMessageFromTg(msg, b.config.Phone))
		if err != nil {
			log.Printf("failed to add new message: %s", err)
		}
		log.Printf("saved message: %v\n", saved)
	}()
	return nil
}

func (b *UsrBot) DeleteMessagesHandler(ctx context.Context, e tg.Entities, update *tg.UpdateDeleteMessages) error {
	go func() {
		wait := b.savingNewMsgs.Load()
		for wait {
			log.Printf("waiting for saving new messages")
			time.Sleep(200 * time.Millisecond)
			wait = b.savingNewMsgs.Load()
		}

		log.Println("delete", update.Messages)
		log.Printf("entities: %+v", e)

		msgs := make(map[dialogs.DialogKey][]*model.Message, len(update.Messages))
		peers := make(map[model.PeerKey]*storage.Peer)
		for _, id := range update.Messages {
			msg, err := b.repo.MessagesRepo.GetNonChannelMsg(ctx, b.config.Phone, id)
			if err != nil {
				log.Printf("failed to get non channel message: %s", err)
				continue
			}
			log.Printf("loaded deleted msg: %+v", msg)
			pkey := msg.PeerKey()
			p, err := FindPeer(ctx, b.peerDB, pkey)
			if err != nil {
				log.Printf("failed to find peer: %s", err)
				continue
			}
			err = b.repo.MessagesRepo.MarkDeletedMessage(ctx, b.config.Phone, pkey, msg.MsgID)
			if errors.Is(err, repository.ErrNoAffectedRows) {
				log.Printf("not found to mark deleted message: %s", err.Error())
			}
			if err != nil {
				log.Printf("failed to mark deleted message: %s", err.Error())
			}

			peers[pkey] = &p
			_, ok := msgs[dialogs.DialogKey(pkey)]
			if !ok {
				msgs[dialogs.DialogKey(pkey)] = []*model.Message{msg}
			} else {
				msgs[dialogs.DialogKey(pkey)] = append(msgs[dialogs.DialogKey(pkey)], msg)
			}
		}
		log.Printf("peers: %+v", peers)
		log.Printf("msgs: %+v", msgs)
		msgToSend, err := templates.DeletedMessagesTemplate(msgs, slices.Collect(maps.Values(peers)))
		if err != nil {
			log.Println(err.Error())
			return
		}
		b.tgbot.SendMessage(ctx, msgToSend, 0)
	}()
	return nil
}
