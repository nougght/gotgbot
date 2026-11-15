package model

import (
	"time"

	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"
)

type (
	Message struct {
		Phone    string `db:"phone"`
		PeerType int    `db:"peer_type"`
		PeerId   int64  `db:"peer_id"`
		FromID   *int64 `db:"from_id"`
		MsgID    int    `db:"msg_id"`

		Out       bool       `db:"out"`
		Content   string     `db:"content"`
		Date      time.Time  `db:"date"`
		DeletedAt *time.Time `db:"deleted_at"`
	}
	PeerKey dialogs.DialogKey
)

const (
	PeerTypeUser = iota
	PeerTypeChat
	PeerTypeChannel
)

func resolvePeerID(peer tg.PeerClass) (id int64, peerType int) {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID, PeerTypeUser
	case *tg.PeerChat:
		return p.ChatID, PeerTypeChat
	case *tg.PeerChannel:
		return p.ChannelID, PeerTypeChannel
	}
	return 0, 0
}

func NewMessageFromTg(msg *tg.Message, phone string) *Message {
	peerID, peerType := resolvePeerID(msg.GetPeerID())
	var fromID *int64
	if fID, ok := msg.GetFromID(); ok {
		fpID, _ := resolvePeerID(fID)
		fromID = &fpID
	}
	return &Message{
		Phone:     phone,
		PeerType:  peerType,
		PeerId:    peerID,
		FromID:    fromID,
		MsgID:     msg.GetID(),
		Out:       msg.GetOut(),
		Content:   msg.Message,
		Date:      time.Unix(int64(msg.GetDate()), 0),
		DeletedAt: nil,
	}
}

func (m *Message) PeerKey() PeerKey {
	return PeerKey{
		Kind: dialogs.PeerKind(m.PeerType),
		ID:   m.PeerId,
	}
}
