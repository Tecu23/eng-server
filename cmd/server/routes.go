// Package main is the entry point of the application
package main

import (
	"net/http"

	"go.uber.org/zap"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	app.Logger.Info("Starting server", zap.String("address", ":"+app.Config.Port))
	if err := http.ListenAndServe(":"+app.Config.Port, nil); err != nil {
		app.Logger.Fatal("ListenAndServe error", zap.Error(err))
	}

	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/ws", app.authenticate(app.handleHealth))

	return mux
}
