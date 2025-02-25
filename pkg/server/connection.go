package server

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/tecu23/eng-server/pkg/messages"
)

type Connection struct {
	ID      uuid.UUID
	ws      *websocket.Conn // The underlying Websocket connection
	hub     *Hub
	send    chan []byte // Buffered channel of outbound messages.
	writeMu sync.Mutex  // Mutex to protect concurrent writes to ws.
}

func NewConnection(ws *websocket.Conn, hub *Hub) *Connection {
	return &Connection{
		ID:   uuid.New(),
		ws:   ws,
		hub:  hub,
		send: make(chan []byte, 256), // buffer3ed for outgoing messages
	}
}

// ReadPump handles inbound messages from the client
func (c *Connection) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	for {
		msgType, msg, err := c.ws.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		// We only handle text
		if msgType == websocket.TextMessage {
			var inbound messages.InboundMessage
			if err := json.Unmarshal(msg, &inbound); err == nil {
				c.hub.inbound <- InboundHubMessage{
					Conn:    c,
					Message: inbound,
				}
			} else {
				log.Println("Failed to parse inbound JSON:", err)
			}
		}
	}
}

// WritePump handles outbound messages to the client
func (c *Connection) WritePump() {
	defer func() {
		c.ws.Close()
	}()

	for {
		message, ok := <-c.send
		if !ok {
			// Channel closed
			log.Println("send channel closed for connection")
			return
		}
		c.writeMu.Lock()
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		c.writeMu.Unlock()

		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}

// SendJSON is a helper for sending JSON to this connection
func (c *Connection) SendJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		return
	}

	c.send <- data
}
