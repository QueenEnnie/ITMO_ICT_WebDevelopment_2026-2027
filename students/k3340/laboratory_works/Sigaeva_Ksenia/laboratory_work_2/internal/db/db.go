package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	assets "homeworkboard"
)

func Open(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(10)
	conn.SetConnMaxLifetime(30 * time.Minute)
	if err = conn.Ping(); err == nil {
		err = InitSchema(conn)
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func InitSchema(conn *sql.DB) error {
	schema, err := assets.Files.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(schema))
	return err
}

func Duplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
