package database

import (
	"context"
	"database/sql"
	_ "embed"

	_ "modernc.org/sqlite" // sqlite3 driver
)

//go:embed schema.sql
var databaseInit string

var initialized bool

func initialize(db *sql.DB) error {
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, databaseInit); err != nil {
		return err
	}

	initialized = true
	return nil
}

func Connect(databaseFilepath string) (Querier, *sql.DB, error) {

	db, err := sql.Open("sqlite", databaseFilepath)
	if err != nil {
		return nil, nil, err
	}

	db.SetMaxOpenConns(1)

	if err := initialize(db); err != nil {
		return nil, nil, err
	}

	return New(db), db, nil
}
