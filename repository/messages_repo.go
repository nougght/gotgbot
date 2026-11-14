package repository

import (
	"context"
	"errors"
	"fmt"
	"gotgbot/model"
	database "gotgbot/repository/db"
	"log"

	"github.com/jackc/pgx/v5"
)

type MessagesRepo struct {
	db database.DB
}

func NewMessagesRepo(db database.DB) *MessagesRepo {
	return &MessagesRepo{
		db: db,
	}
}

func (s *MessagesRepo) GetPeerMessage(ctx context.Context, phone string, peerType int, peerID int64, msgID int64) (res *model.Message, err error) {
	query := `
		SELECT * FROM messages 
		WHERE phone = $1
		AND peer_type = $2
		AND peer_id = $3
		AND msg_id = $4;
		`

	rows, err := s.db.Query(ctx, query, phone, peerType, peerID, msgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err = pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[model.Message])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to collect msgs: %w", err)
	}

	return res, nil
}

func (s *MessagesRepo) GetPeerMessages(ctx context.Context, phone string, peerType int, peerID int64) (res []*model.Message, err error) {
	query := `
		SELECT * FROM messages 
		WHERE phone = $1
		AND peer_type = $2
		AND peer_id = $3;
		`

	rows, err := s.db.Query(ctx, query, phone, peerType, peerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[model.Message])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to collect msgs: %w", err)
	}

	return res, nil
}

func (s *MessagesRepo) AddMessage(ctx context.Context, msg *model.Message) (*model.Message, error) {
	query := `
		INSERT INTO messages(
			phone,
			peer_id,
			from_id,
			msg_id,
			out,
			content,
			date,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (phone, peer_type, peer_id, msg_id) DO NOTHING;`

	res, err := s.db.Exec(ctx, query,
		msg.Phone,
		msg.PeerId,
		msg.FromID,
		msg.MsgID,
		msg.Out,
		msg.Content,
		msg.Date,
		msg.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add msg: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, ErrNoAffectedRows
	}
	return msg, nil
}

func splitMsgsList(msgs []*model.Message) [][]any {
	res := make([][]any, 9)
	for i := range res {
		res[i] = make([]any, 0, len(msgs))
	}
	for _, msg := range msgs {
		res[0] = append(res[0], msg.Phone)
		res[1] = append(res[1], msg.PeerType)
		res[2] = append(res[2], msg.PeerId)
		res[3] = append(res[3], msg.FromID)
		res[4] = append(res[4], msg.MsgID)
		res[5] = append(res[5], msg.Out)
		res[6] = append(res[6], msg.Content)
		res[7] = append(res[7], msg.Date)
		res[8] = append(res[8], msg.DeletedAt)
	}
	return res
}

func (s *MessagesRepo) BatchAddMessages(ctx context.Context, msgs []*model.Message) error {
	query := `
		INSERT INTO messages(
			phone,
			peer_type,
			peer_id,
			from_id,
			msg_id,
			out,
			content,
			date,
			deleted_at
		)
		SELECT * FROM unnest($1::varchar[], $2::int[], $3::bigint, $4::bigint, $5::bigint, 
			$6::boolean, $7::text, $8::timestamptz[], $9::timestamptz[])
		ON CONFLICT (phone, peer_type, peer_id, msg_id) DO NOTHING;`
	args := splitMsgsList(msgs)
	res, err := s.db.Exec(ctx, query, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9])
	if err != nil {
		return fmt.Errorf("failed to batch add msg: %w", err)
	}

	if res.RowsAffected() != int64(len(msgs)) {
		log.Printf("not equal batch size: %d != %d", res.RowsAffected(), int64(len(msgs)))
	}
	return nil
}
