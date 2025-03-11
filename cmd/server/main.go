// Package main is the entry point of the application
package main

import (
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tecu23/eng-server/internal/auth"
	"github.com/tecu23/eng-server/pkg/config"
	"github.com/tecu23/eng-server/pkg/engine"
	"github.com/tecu23/eng-server/pkg/events"
	"github.com/tecu23/eng-server/pkg/manager"
	"github.com/tecu23/eng-server/pkg/repository"
	"github.com/tecu23/eng-server/pkg/server"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		path := os.Getenv("FRONTEND_PATH")
		return path == r.Header.Get("Origin")
	},
}

// App encapsulates global dependencies
type application struct {
	Auth      *auth.APIKeyAuth
	Logger    *zap.Logger
	Config    *config.Config
	Publisher *events.Publisher
	Hub       *server.Hub
	Server    *http.Server

	StartTime time.Time
}

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	config := &config.Config{
		Debug: *debug,
		Port:  *port,
	}

	// Initialize logger
	logger := initLogger(config.Debug)
	defer logger.Sync()

	err := godotenv.Load()
	if err != nil {
		logger.Fatal("loading env error", zap.Error(err))
	}

	// Initialize event publisher
	publisher := events.NewPublisher()

	// Initialize repository
	repository := repository.NewInMemoryRepository(logger)

	// Initlialize engine pool
	enginePool := engine.NewEnginePool(os.Getenv("ENGINE_PATH"), 5, logger)
	if err := enginePool.Initialize(); err != nil {
		logger.Fatal("initialize engine error", zap.Error(err))
	}

	// Initialize game manager
	gm := manager.NewManager(repository, enginePool, logger, publisher)

	hub := server.NewHub(gm, publisher, logger)

	var authKeys []string

	if envAPIKeys := os.Getenv("API_KEYS"); envAPIKeys != "" {
		// Split comma-separated list of API keys
		keys := strings.Split(envAPIKeys, ",")
		for i, key := range keys {
			keys[i] = strings.TrimSpace(key)
		}
		authKeys = keys
	}

	app := &application{
		Auth:      auth.NewAPIKeyAuth(authKeys),
		Logger:    logger,
		Config:    config,
		Hub:       hub,
		Publisher: publisher,
		StartTime: time.Now(),
	}

	go app.Hub.Run()

	err = app.serve()
	if err != nil {
		logger.Fatal("error serving", zap.Error(err))
	}
}

func initLogger(debug bool) *zap.Logger {
	var cfg zap.Config
	if debug {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	return logger
}

// Shutdown cleans up resources
func (app *application) Shutdown() {
	// Shut down hub
	if app.Hub != nil {
		app.Hub.Shutdown()
	}

	app.Logger.Info("All components shut down successfully")
}
