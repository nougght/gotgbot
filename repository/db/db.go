package database

import (
	"context"
	"fmt"
	"io/fs"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST,required"`
	Port     string `env:"POSTGRES_PORT,required"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	DBName   string `env:"POSTGRES_DB,required"`
	SSLMode  string `env:"POSTGRES_SSLMODE,required"`
}

func (c *PostgresConfig) ConnString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	// begin real or pseudo transaction
	Begin(ctx context.Context) (pgx.Tx, error)
}

func RunMigrations(migrationFS fs.FS, pool *pgxpool.Pool) error {
	goose.SetBaseFS(migrationFS)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		_ = db.Close()
	}()

	if err := goose.Up(db, "repository/migrations"); err != nil {
		return fmt.Errorf("migrate up failed: %w", err)
	}

	return nil
}

func connect(ctx context.Context, config *PostgresConfig) (*pgxpool.Pool, error) {
	// TODO: add retrier
	pool, err := pgxpool.New(ctx, config.ConnString())
	if err != nil {
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}
	return pool, nil
}

func ConnectAndMigrate(ctx context.Context, config *PostgresConfig, migrationFS fs.FS) (*pgxpool.Pool, error) {
	pool, err := connect(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(migrationFS, pool); err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}
	log.Println("successful migrations")
	return pool, nil
}
