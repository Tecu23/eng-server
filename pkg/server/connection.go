package server

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/events"
)

type Connection struct {
	ID      uuid.UUID
	ws      *websocket.Conn // The underlying Websocket connection
	hub     *Hub
	send    chan []byte // Buffered channel of outbound messages.
	writeMu sync.Mutex  // Mutex to protect concurrent writes to ws.

	publisher *events.Publisher
	logger    *zap.Logger
}

func NewConnection(
	ws *websocket.Conn,
	hub *Hub,
	publisher *events.Publisher,
	logger *zap.Logger,
) *Connection {
	return &Connection{
		ID:        uuid.New(),
		ws:        ws,
		hub:       hub,
		send:      make(chan []byte, 256), // buffered for outgoing messages
		publisher: publisher,
		logger:    logger,
	}
}

// ReadPump handles inbound messages from the client
func (c *Connection) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	// Publish connection closed event
	c.publisher.Publish(events.Event{
		Type: events.EventConnectionClosed,
		Payload: map[string]string{
			"connection_id": c.ID.String(),
		},
	})

	for {
		msgType, msg, err := c.ws.ReadMessage()
		if err != nil {
			c.logger.Error("read error", zap.Error(err))
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
				c.logger.Error("Failed to parse inbound JSON", zap.Error(err))
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
			c.logger.Info(
				"Send channel closed for connection",
				zap.String("connection_id", c.ID.String()),
			)
			return
		}
		c.writeMu.Lock()
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		c.writeMu.Unlock()
		if err != nil {
			c.logger.Error("write error", zap.Error(err))
			return
		}
	}
}

// SendJSON is a helper for sending JSON to this connection
func (c *Connection) SendJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		c.logger.Error("Error marshaling JSON", zap.Error(err))
		return
	}

	c.send <- data
}
