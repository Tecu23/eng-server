// Package main is the entry point of the application
package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", app.handleHealth)

	// For serving all files in the docs directory
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs"))))

	mux.HandleFunc("/ws", app.authenticate(app.handleHealth))

	app.Logger.Info("Routes configured successfully")

	return mux
}
