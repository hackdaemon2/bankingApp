package main

import (
	"bankingApp/configuration" // nolint
	"fmt"
	"net/http"
)

const bytes = 1 << 20

func main() {
	app := configuration.NewApp()
	config := app.Configuration
	routeHandler := app.RouteHandler(config)
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", config.ServerPort()),
		Handler:        routeHandler,
		MaxHeaderBytes: bytes,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
