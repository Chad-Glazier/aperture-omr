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
	Mode                  RuntimeEnvironment // Test, Development, or Production
	Host                  string             // The hostname for the HTTP server.
	Port                  string             // The port for the HTTP server.
	Version               string             // The version of the OMR service.
	DatabaseHost          string
	DatabasePort          string
	DatabaseUser          string
	DatabasePassword      string
	DatabaseName          string
	GarageAccessKeyId     string
	GarageSecretAccessKey string
	GarageEndpointUrl     string
	GarageRegion          string
	GarageBucketName      string
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
		Mode = Test
		Host = "localhost"
		Port = "3000"
		Version = "test"

	case "DEVELOPMENT":
		Mode = Development
		Host = "localhost"
		Port = getenv("OMR_PORT", "3000")
		Version = getenv("OMR_VERSION", "0.0.1")
		DatabaseHost = getenv("POSTGRES_HOST", "localhost")
		DatabasePort = getenv("POSTGRES_PORT", "5432")
		DatabaseUser = getenv("POSTGRES_USER", "test_user")
		DatabasePassword = getenv("POSTGRES_PASSWORD", "pass")
		DatabaseName = getenv("POSTGRES_DB", "test_database")
		GarageAccessKeyId = getenv(
			"GARAGE_ACCESS_KEY_ID",
			"GK226e804ce6278cc5d0ebc0a6")
		GarageSecretAccessKey = getenv(
			"GARAGE_SECRET_ACCESS_KEY",
			"eeafdc5181ed85548bc2b61da795e86a2d118dc9807b52b56227e15b84abdfb2")
		GarageEndpointUrl = getenv(
			"GARAGE_ENDPOINT_URL",
			"http://garage-store:3900/")
		GarageRegion = getenv(
			"GARAGE_REGION",
			"garage")
		GarageBucketName = getenv(
			"GARAGE_BUCKET_NAME",
			"capstone-storage")

	case "PRODUCTION":
		Mode = Production
		Host = "localhost"
		Port = mustGetenv("OMR_PORT")
		Version = getenv("OMR_VERSION", "0.0.1")
		DatabaseHost = mustGetenv("POSTGRES_HOST")
		DatabasePort = mustGetenv("POSTGRES_PORT")
		DatabaseUser = mustGetenv("POSTGRES_USER")
		DatabasePassword = mustGetenv("POSTGRES_PASSWORD")
		DatabaseName = mustGetenv("POSTGRES_DB")
		GarageAccessKeyId = mustGetenv("GARAGE_ACCESS_KEY_ID")
		GarageSecretAccessKey = mustGetenv("GARAGE_SECRET_ACCESS_KEY")
		GarageEndpointUrl = mustGetenv("GARAGE_ENDPOINT_URL")
		GarageRegion = mustGetenv("GARAGE_REGION")
		GarageBucketName = mustGetenv("GARAGE_BUCKET_NAME")
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
	return Mode == Test
}

// Logs a message indicating the runtime mode (development, production, or
// test).
func LogMode() {
	switch Mode {
	case Test:
		slog.Info("starting in test mode")
	case Development:
		slog.Info("starting in development mode")
	case Production:
		slog.Info("starting in production mode")
	}
}
