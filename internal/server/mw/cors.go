/*
Implements middleware for the HTTP server.
*/
package mw

import (
	"github.com/rs/cors"
)

// Middleware to configure CORS.
var Cors = cors.Default().Handler
