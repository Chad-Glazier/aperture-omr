package db

import (
	"context"
	"fmt"
	"ubc/team15/config"
	"ubc/team15/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
)

func Connect() (*sqlc.Queries, error) {
	ctx := context.Background()

	// Check the link below for a description of how Postgres connection 
	// strings work: 
	// https://www.geeksforgeeks.org/postgresql/postgresql-connection-string/
	conn, err := pgx.Connect(ctx, fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s",
		config.DATABASE_USER,
		config.DATABASE_PASSWORD,
		config.DATABASE_HOST,
		config.DATABASE_PORT,
		config.DATABASE_NAME,
	))
	if err != nil {
		return nil, err
	}

	return sqlc.New(conn), nil
}
