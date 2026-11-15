package repository

import (
	"gotgbot/config"
	database "gotgbot/repository/db"
)

type Repository struct {
	SessionRepo  *SessionStorage
	MessagesRepo *MessagesRepo
}

func NewRepository(db database.DB, cfg *config.Config) *Repository {
	return &Repository{
		SessionRepo:  NewSessionStorage(db, cfg),
		MessagesRepo: NewMessagesRepo(db),
	}
}
