package usrbot

import (
	"context"
	"errors"
	"fmt"
	"gotgbot/model"
	"gotgbot/repository"
	"log"
	"slices"
	"time"

	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/tg"
	"golang.org/x/time/rate"
)

type (
	dialogsList struct {
		users    []*tg.InputPeerUser
		chats    []*tg.InputPeerChat
		channels []*tg.InputPeerChannel
	}
	peerFinder struct {
		store storage.PeerStorage
	}
)

func (b *UsrBot) FindUserByID(ctx context.Context, id int64) (*tg.User, error) {
	peer, err := storage.FindPeer(ctx, b.peerDB, &tg.PeerUser{UserID: id})
	if err != nil {
		return nil, fmt.Errorf("failed find user by id: %w", err)
	}
	return peer.User, nil
}

func (d *dialogsList) AllTypes() []tg.InputPeerClass {
	res := make([]tg.InputPeerClass, 0, len(d.users)+len(d.chats)+len(d.channels))
	for _, user := range d.users {
		res = append(res, user)
	}
	for _, chat := range d.chats {
		res = append(res, chat)
	}
	//for _, channel := range d.channels {
	//	res = append(res, channel)
	//}
	return res
}

func DialogsCount(ctx context.Context, api *tg.Client) (int, error) {
	return query.GetDialogs(api).
		BatchSize(1).Count(ctx)
}

func GetDialogs(ctx context.Context, client *tg.Client) (res *dialogsList, err error) {
	log.Println("start collect dialogs")
	elems, err := query.GetDialogs(client).
		BatchSize(100).Collect(ctx)

	log.Println("finish collect dialogs")
	res = &dialogsList{
		users:    make([]*tg.InputPeerUser, 0, len(elems)),
		chats:    make([]*tg.InputPeerChat, 0, len(elems)),
		channels: make([]*tg.InputPeerChannel, 0, len(elems)),
	}
	for _, elem := range elems {
		switch p := elem.Peer.(type) {
		case *tg.InputPeerUser:
			res.users = append(res.users, p)
		case *tg.InputPeerChat:
			res.chats = append(res.chats, p)
		case *tg.InputPeerChannel:
			res.channels = append(res.channels, p)
		}
	}
	return res, err
}

func (b *UsrBot) SaveMessagesByPeers(ctx context.Context, client *tg.Client, peers []tg.InputPeerClass) error {
	ctx = context.Background()
	go func() {
		time.Sleep(time.Second)
		res := make([]*model.Message, 0, 1000)
		rl := rate.NewLimiter(rate.Every(time.Second), 1)
		for _, peer := range peers {
			q := query.Messages(client)
			it := q.GetHistory(peer).BatchSize(100).Iter()
			batchLen, err := it.Total(ctx)
			if err != nil {
				log.Printf("failed to get msgs total: %s", err.Error())
				continue
			}
			res = slices.Grow(res, len(res)+batchLen)
			for it.Next(ctx) {
				msg := it.Value().Msg
				switch m := msg.(type) {
				case *tg.Message:
					res = append(res, model.NewMessageFromTg(m, b.config.Phone))
				case *tg.MessageService:
					// TODO: add service msgs
				}
				if len(res) >= 100 {
					log.Printf("saving msgs batch of peer: %#v", peer)
					log.Printf("messages len: %d", len(res))
					err = b.repo.MessagesRepo.BatchAddMessages(ctx, res)
					res = res[:0]
					if errors.Is(err, repository.ErrNoAffectedRows) {
						log.Printf("msgs already exist: %s", err.Error())
						break
					}
					if err != nil {
						log.Printf("failed to save msgs batch of peer: %s", err.Error())
					}
					rl.Wait(ctx)
				}
			}
			if err := it.Err(); err != nil {
				log.Printf("failed to iterate msgs: %s", err.Error())
				continue
			}
			rl.Wait(ctx)
			time.Sleep(100 * time.Millisecond)
			log.Printf("iterated %d messages of peer %v", batchLen, peer)
		}
		// save remaining msgs
		if len(res) > 0 {
			err := b.repo.MessagesRepo.BatchAddMessages(ctx, res)
			res = res[:0]
			if errors.Is(err, repository.ErrNoAffectedRows) {
				log.Printf("msgs already exist: %s", err.Error())
			}
			if err != nil {
				log.Printf("failed to save msgs batch of peer: %s", err.Error())
			}
		}
		log.Println("finish save messages by peers")
	}()
	return nil
}

func (b *UsrBot) ListUserMsgs(ctx context.Context, userID int64, limit, offset int64) ([]*model.Message, error) {
	msgs, err := b.repo.MessagesRepo.GetPeerMessages(ctx, b.config.Phone, model.PeerTypeUser, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// FindPeer finds peer using given storage
func FindPeer(ctx context.Context, s storage.PeerStorage, key model.PeerKey) (storage.Peer, error) {

	return s.Find(ctx, storage.PeerKey{
		Kind: key.Kind,
		ID:   key.ID,
	})
}
