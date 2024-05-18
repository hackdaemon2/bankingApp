package main

import (
	"bankingApp/configuration" // nolint
	"fmt"
	"net/http"
)

func main() {
	app := configuration.NewApp()
	config := app.Configuration
	routeHandler := app.RouteHandler(config)
	const bytes = 1 << 20
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
