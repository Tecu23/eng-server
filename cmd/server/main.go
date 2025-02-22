// Package main is the entry point of the application
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

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

func main() {
	gm := game.NewManager()

	hub := server.NewHub(gm)
	go hub.Run()

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Server is up and running!")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading to websocket:", err)
			return
		}

		conn := server.NewConnection(ws, hub)
		hub.Register(conn)

		go conn.WritePump()
		go conn.ReadPump()
	})

	log.Println("Starting serve on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
