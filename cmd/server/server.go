package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Run starts the http server and handles graceful shutdown
func (app *application) serve() error {
	app.Server = &http.Server{
		Addr:         ":" + app.Config.Port,
		Handler:      app.routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		// Set up signal handling for graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Wait for shutdown signal
		s := <-quit
		app.Logger.Info("Shutting down server", zap.String("signal", s.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		err := app.Server.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.Logger.Error("Server forced to shutdown", zap.Error(err))

		// Shut down components
		app.Shutdown()
		shutdownError <- nil
	}()

	app.Logger.Info("Starting server", zap.String("address", app.Server.Addr))

	if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.Logger.Fatal("Server error", zap.Error(err))
	}

	err := <-shutdownError
	if err != nil {
		return err
	}

	app.Logger.Info("Server stopped gracefully")
	return nil
}
