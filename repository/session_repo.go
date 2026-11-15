package repository

import (
	"context"
	"errors"
	"gotgbot/config"
	database "gotgbot/repository/db"
	"log"

	"github.com/gotd/td/session"
	"github.com/jackc/pgx/v5"
)

type SessionStorage struct {
	db  database.DB
	cfg *config.Config
}

func NewSessionStorage(db database.DB, cfg *config.Config) *SessionStorage {
	return &SessionStorage{
		db:  db,
		cfg: cfg,
	}
}

func (s *SessionStorage) LoadSession(ctx context.Context) (res []byte, err error) {
	query := `
		SELECT data FROM sessions WHERE phone = $1;`

	err = s.db.QueryRow(ctx, query, s.cfg.Phone).Scan(&res)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("not found session")
		return nil, session.ErrNotFound
	}
	log.Printf("loaded  session: %s", res)
	return res, err
}

func (s *SessionStorage) StoreSession(ctx context.Context, data []byte) error {
	query := `
		INSERT INTO sessions(phone, data) 
		VALUES ($1, $2)
		ON CONFLICT (phone) DO UPDATE SET
          	data = EXCLUDED.data;`

	_, err := s.db.Exec(ctx, query, s.cfg.Phone, data)

	return err
}
