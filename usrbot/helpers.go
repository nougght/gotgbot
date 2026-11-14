package usrbot

import (
	"context"
	"log"

	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/tg"
)

type (
	dialogsList struct {
		users    []*tg.InputPeerUser
		chats    []*tg.InputPeerChat
		channels []*tg.InputPeerChannel
	}
)
func (d *dialogsList) AllTypes() []tg.InputPeerClass {
	res := make([]tg.InputPeerClass, 0, len(d.users) + len(d.chats) + len(d.channels))
	for _, user := range d.users {
		res = append(res, user)
	}
	for _, chat := range d.chats {
		res = append(res, chat)
	}
	for _, channel := range d.channels {
		res = append(res, channel)
	}
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

func GetDialogMessages(ctx context.Context, client *tg.Client, list dialogsList) {
	for query.Messages(client).GetHistory(list.users[0])
}
