package repository

import (
	"context"
	"errors"
	database "gotgbot/repository/db"

	"github.com/gotd/td/session"
	"github.com/jackc/pgx/v5"
)

type SessionStorage struct {
	db    database.DB
	phone string
}

func NewSessionStorage(db database.DB, phone string) *SessionStorage {
	return &SessionStorage{
		db:    db,
		phone: phone,
	}
}

func (s *SessionStorage) LoadSession(ctx context.Context) (res []byte, err error) {
	query := `
		SELECT data FROM sessions WHERE phone = $1;`

	err = s.db.QueryRow(ctx, query, s.phone).Scan(&res)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, session.ErrNotFound
	}
	return res, err
}
func (s *SessionStorage) StoreSession(ctx context.Context, data []byte) error {
	query := `
		INSERT INTO sessions(phone, data) 
		VALUES ($1, $2)
		ON CONFLICT (phone) DO UPDATE SET
          	data = EXCLUDED.data;`

	_, err := s.db.Exec(ctx, query, s.phone, data)

	return err
}
