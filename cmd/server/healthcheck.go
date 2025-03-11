// Package main is the entry point of the application
package main

import (
	"fmt"
	"net/http"
	"time"
)

// handleHealth handles the GET /health endpoint
func (app *application) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","uptime":"%s"}`, time.Since(app.StartTime))
}
