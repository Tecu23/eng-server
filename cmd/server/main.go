// Package main is the entry point of the application
package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tecu23/eng-server/pkg/config"
	"github.com/tecu23/eng-server/pkg/game"
	"github.com/tecu23/eng-server/pkg/server"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(_ *http.Request) bool {
		return true // Allow all connections for now
	},
}

// App encapsulates global dependencies
type application struct {
	Logger *zap.Logger
	Config *config.Config
	Hub    *server.Hub
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

	gm := game.NewManager(logger)
	hub := server.NewHub(gm, logger)

	app := &application{
		Logger: logger,
		Config: config,
		Hub:    hub,
	}

	go app.Hub.Run()

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Server is up and running!")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			app.Logger.Error("Error upgrading to websocket:", zap.Error(err))
			return
		}

		conn := server.NewConnection(ws, hub, app.Logger)
		app.Hub.Register(conn)

		go conn.WritePump()
		go conn.ReadPump()
	})

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
