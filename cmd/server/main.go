// Package main is the entry point of the application
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tecu23/eng-server/pkg/config"
	"github.com/tecu23/eng-server/pkg/events"
	"github.com/tecu23/eng-server/pkg/game"
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
	Logger    *zap.Logger
	Config    *config.Config
	Publisher *events.Publisher
	Hub       *server.Hub
}

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	config := &config.Config{
		Debug: *debug,
		Port:  *port,
	}

	logger := initLogger(config.Debug)
	defer logger.Sync()

	publisher := events.NewPublisher()

	gm := game.NewManager(logger, publisher)
	hub := server.NewHub(gm, publisher, logger)

	app := &application{
		Logger:    logger,
		Config:    config,
		Hub:       hub,
		Publisher: publisher,
	}

	go app.Hub.Run()

	err := godotenv.Load()
	if err != nil {
		app.Logger.Fatal("loading env error", zap.Error(err))
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Server is up and running!")
	})

	http.HandleFunc("/ws", app.wsHandler)
	// http.HandleFunc("/ws", app.authenticate(http.HandlerFunc(app.wsHandler)))

	app.Logger.Info("Starting server", zap.String("address", ":"+app.Config.Port))
	if err := http.ListenAndServe(":"+app.Config.Port, nil); err != nil {
		app.Logger.Fatal("ListenAndServe error", zap.Error(err))
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

func (app *application) wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Logger.Error("Error upgrading to websocket:", zap.Error(err))
		return
	}

	conn := server.NewConnection(ws, app.Hub, app.Publisher, app.Logger)
	app.Hub.Register(conn)

	go conn.WritePump()
	go conn.ReadPump()
}
