// Package main provides a system that receives log requests and forwards them to various
// configured recipients.
package main

import (
	"os"
	"strings"

	"github.com/kentquirk/stringset/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var validTokens *stringset.StringSet

// this checks the validity of the token passed in to the keyAuth middleware
func tokenValidator(tok string, ctx echo.Context) (bool, error) {
	return validTokens.Contains(tok), nil
}

func main() {
	tokens := strings.Split(os.Getenv("LOGJAM_TOKENS"), ",")
	validTokens = stringset.New().Add(tokens...)

	// Echo instance
	e := echo.New()

	// Middleware for token authentication
	e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:x-logjam-token",
		Validator: tokenValidator,
	}))
	// Middleware for logging (useful for testing but it's kind of the point of this system)
	// e.Use(middleware.Logger())
	// middleware for crash handling
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", err400)
	e.GET("/doc", doc)
	e.GET("/health", health)
	e.PUT("/log", logSinglePut)
	e.POST("/log", logSinglePost)
	e.POST("/multi", logMulti)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
