// Package main is the entry point of the application
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for now
	},
}

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Server is up and running!")
	})

	// WS route
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading to websocket:", err)
			return
		}

		handleConnection(conn)
	})

	log.Println("Starting serve on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}

func handleConnection(conn *websocket.Conn) {
	defer conn.Close()
	log.Println("New Websocket connection established!")

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Printf("Received message: %s\n", msg)

		if err := conn.WriteMessage(msgType, msg); err != nil {
			log.Println("Write error:", err)
			break
		}
	}

	log.Println("Websocket connection closed.")
}
