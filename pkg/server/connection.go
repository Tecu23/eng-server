package server

import (
	"log"

	"github.com/gorilla/websocket"
)

type Connection struct {
	ws   *websocket.Conn
	hub  *Hub
	send chan []byte
}

func NewConnection(ws *websocket.Conn, hub *Hub) *Connection {
	return &Connection{
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
			inbound := InboundMessage{
				Conn:    c,
				Payload: msg,
			}

			c.hub.inbound <- inbound
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
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("write error:", err)
			return
		}
	}
}
