// Package main is the entry point of the application
package main

import (
	"net/http"

	"go.uber.org/zap"
)

func (app *application) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		apiKey := r.Header.Get("X-Api-Key")

		if app.Auth.IsValidKey(apiKey) {
			next.ServeHTTP(w, r)
			return
		}

		app.Logger.Warn(
			"Authentication failed",
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)
		w.Header().Set("WWW-Authenticate", "APIKey")
		http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
	})
}
