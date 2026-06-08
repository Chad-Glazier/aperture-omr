/*
This package contains configuration constants.
*/
package config

//
// This file manages configuration constants.
//

import (
	"log/slog"
	"os"
)

var (
	HOST              string // The hostname for the HTTP server.
	PORT              string // The port for the HTTP server.
	VERSION           string // The semantic version of the OMR service.
	DATABASE_HOST     string
	DATABASE_PORT     string
	DATABASE_USER     string
	DATABASE_PASSWORD string
	DATABASE_NAME     string
)

func init() {
	HOST = "localhost"
	PORT = "3000"
	VERSION = "0.0.1"
	DATABASE_HOST = MustGetenv("POSTGRES_HOST")
	DATABASE_PORT = MustGetenv("POSTGRES_PORT")
	DATABASE_USER = MustGetenv("POSTGRES_USER")
	DATABASE_PASSWORD = MustGetenv("POSTGRES_PASSWORD")
	DATABASE_NAME = MustGetenv("POSTGRES_DB")
}

// Returns an environment variable's value if it exists and exits the program
// otherwise.
func MustGetenv(key string) string {
	value, isDefined := os.LookupEnv(key)
	if !isDefined {
		slog.Error("expected environment variable " + key + " to be defined")
		os.Exit(1)
	}
	return value
}

// Returns an environment variable's value if it exists, returning the
// specified default otherwise.
func MayGetenv(key string, fallback string) string {
	value, isDefined := os.LookupEnv(key)
	if !isDefined {
		return fallback
	}
	return value
}
