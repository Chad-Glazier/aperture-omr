package database

import (
	"context"
	"fmt"
	"ubco-team15/omr/config"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	Query Querier
	conn  *pgx.Conn
}

// Attempts to connect to the database. If the connection attempt was
// successful then nil is returned, otherwise an error is returned.
func CheckConnection() error {
	if config.TestMode() {
		return nil
	}

	conn, err := Open()
	if err != nil {
		return err
	}
	return Close(conn)
}

// Opens a database connection. Remember to close it when you're done with it.
func Open() (*DB, error) {
	if config.TestMode() {
		return GetMockDB(), nil
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s",
		config.DatabaseUser,
		config.DatabasePassword,
		config.DatabaseHost,
		config.DatabasePort,
		config.DatabaseName,
	))
	if err != nil {
		return nil, err
	}

	db := DB{
		Query: sqlc.New(conn),
		conn:  conn,
	}
	return &db, nil
}

// Closes a database connection.
func Close(db *DB) error {
	if config.TestMode() {
		return nil
	}

	ctx := context.Background()
	return db.conn.Close(ctx)
}
