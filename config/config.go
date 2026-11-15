package config

import (
	database "gotgbot/repository/db"
	"log"

	env "github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppID    int    `env:"APP_ID,required"`
	AppHash  string `env:"APP_HASH,required"`
	Phone    string `env:"PHONE,required"`
	BotToken string `env:"BOT_TOKEN,required"`
	Postgres database.PostgresConfig
}

func ReadConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println(err.Error())
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
