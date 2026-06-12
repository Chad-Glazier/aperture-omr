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

type RuntimeEnvironment int

const (
	Test RuntimeEnvironment = iota
	Development
	Production
)

var (
	MODE                     RuntimeEnvironment // Test, Development, or Production
	HOST                     string             // The hostname for the HTTP server.
	PORT                     string             // The port for the HTTP server.
	VERSION                  string             // The version of the OMR service.
	DATABASE_HOST            string
	DATABASE_PORT            string
	DATABASE_USER            string
	DATABASE_PASSWORD        string
	DATABASE_NAME            string
	GARAGE_ACCESS_KEY_ID     string
	GARAGE_SECRET_ACCESS_KEY string
	GARAGE_ENDPOINT_URL      string
	GARAGE_REGION            string
	GARAGE_BUCKET_NAME       string
)

func init() {

	runtimeEnv, isDefined := os.LookupEnv("OMR_MODE")
	if !isDefined {
		// try checking command-line flags.
		for _, arg := range os.Args[1:] {
			switch arg {
			case "--production", "--prod":
				runtimeEnv = "PRODUCTION"
			case "--development", "--dev":
				runtimeEnv = "DEVELOPMENT"
			case "--test":
				runtimeEnv = "TEST"
			}
		}
	}

	// If the mode is not specified by the environment or the command-line
	// flags, then we default to test mode and log a warning.
	if runtimeEnv == "" {
		slog.Warn(
			"environment variable OMR_MODE not found; " +
				"defaulting to test mode",
		)
		runtimeEnv = "TEST"
	}

	switch runtimeEnv {
	case "TEST":
		MODE = Test
		HOST = "localhost"
		PORT = "3000"
		VERSION = "test"

	case "DEVELOPMENT":
		MODE = Development
		HOST = "localhost"
		PORT = getenv("OMR_PORT", "3000")
		VERSION = getenv("OMR_VERSION", "0.0.1")
		DATABASE_HOST = getenv("POSTGRES_HOST", "localhost")
		DATABASE_PORT = getenv("POSTGRES_PORT", "5432")
		DATABASE_USER = getenv("POSTGRES_USER", "test_user")
		DATABASE_PASSWORD = getenv("POSTGRES_PASSWORD", "pass")
		DATABASE_NAME = getenv("POSTGRES_DB", "test_database")
		GARAGE_ACCESS_KEY_ID = getenv(
			"GARAGE_ACCESS_KEY_ID",
			"GK226e804ce6278cc5d0ebc0a6")
		GARAGE_SECRET_ACCESS_KEY = getenv(
			"GARAGE_SECRET_ACCESS_KEY",
			"eeafdc5181ed85548bc2b61da795e86a2d118dc9807b52b56227e15b84abdfb2")
		GARAGE_ENDPOINT_URL = getenv(
			"GARAGE_ENDPOINT_URL",
			"http://garage-store:3900/")
		GARAGE_REGION = getenv(
			"GARAGE_REGION",
			"garage")
		GARAGE_BUCKET_NAME = getenv(
			"GARAGE_BUCKET_NAME",
			"capstone-storage")

	case "PRODUCTION":
		MODE = Production
		HOST = "localhost"
		PORT = mustGetenv("OMR_PORT")
		VERSION = getenv("OMR_VERSION", "0.0.1")
		DATABASE_HOST = mustGetenv("POSTGRES_HOST")
		DATABASE_PORT = mustGetenv("POSTGRES_PORT")
		DATABASE_USER = mustGetenv("POSTGRES_USER")
		DATABASE_PASSWORD = mustGetenv("POSTGRES_PASSWORD")
		DATABASE_NAME = mustGetenv("POSTGRES_DB")
		GARAGE_ACCESS_KEY_ID = mustGetenv("GARAGE_ACCESS_KEY_ID")
		GARAGE_SECRET_ACCESS_KEY = mustGetenv("GARAGE_SECRET_ACCESS_KEY")
		GARAGE_ENDPOINT_URL = mustGetenv("GARAGE_ENDPOINT_URL")
		GARAGE_REGION = mustGetenv("GARAGE_REGION")
		GARAGE_BUCKET_NAME = mustGetenv("GARAGE_BUCKET_NAME")
	}

}

// Returns an environment variable's value if it exists and exits the program
// otherwise.
func mustGetenv(key string) string {
	value, isDefined := os.LookupEnv(key)
	if !isDefined {
		slog.Error("expected environment variable " + key + " to be defined")
		os.Exit(1)
	}
	return value
}

// Returns an environment variable's value if it exists, returning the
// specified default otherwise. If the default is used a warning is logged.
func getenv(key string, fallback string) string {
	value, isDefined := os.LookupEnv(key)
	if !isDefined {
		slog.Warn(
			"environment variable " + key + " not found; " +
				"defaulting to " + fallback,
		)
		return fallback
	}
	return value
}

// Returns true if and only if the runtime is in testing mode.
func TestMode() bool {
	return MODE == Test
}

// Logs a message indicating the runtime mode (development, production, or
// test).
func LogMode() {
	switch MODE {
	case Test:
		slog.Info("starting in test mode")
	case Development:
		slog.Info("starting in development mode")
	case Production:
		slog.Info("starting in production mode")
	}
}
