// Package main is the entry point of the application
package main

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/tecu23/eng-server/pkg/server"
)

// handleWebSocket handles WebSocket connections
func (app *application) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	// Create and register connection
	conn := server.NewConnection(ws, app.Hub, app.Publisher, app.Logger)
	app.Hub.Register(conn)

	app.Logger.Info("WebSocket connection established",
		zap.String("remote_addr", r.RemoteAddr))

	// Start connection read/write goroutines
	go conn.WritePump()
	go conn.ReadPump()
}
