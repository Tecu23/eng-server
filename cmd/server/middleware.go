package main

import (
	"net/http"

	"go.uber.org/zap"
)

func (app *application) authenticate(next http.HandlerFunc) http.HandlerFunc {
	TestApiKey := "test_api_key"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiKey string

		if apiKey = r.Header.Get("X-Api-Key"); apiKey != TestApiKey {
			app.Logger.Error("bad api key", zap.String(apiKey, ""))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
