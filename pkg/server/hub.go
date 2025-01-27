package server

import (
	"fmt"

	"github.com/tecu23/eng-server/pkg/game"
)

type InboundMessage struct {
	Conn    *Connection // who sent it
	Payload []byte      // raw JSON or text
}

type Hub struct {
	connections map[*Connection]bool // Registered connections

	register   chan *Connection // Incoming registration
	unregister chan *Connection // Incoming unregistration

	inbound chan InboundMessage // Channel or inbound messages that the hub might route or broadcast

	broadcast chan []byte // Channel to broadcast to everyone

	gameSessions map[string]*game.GameSession // Store all game sessions in a map
}

func NewHub() *Hub {
	return &Hub{
		connections:  make(map[*Connection]bool),
		register:     make(chan *Connection),
		unregister:   make(chan *Connection),
		inbound:      make(chan InboundMessage),
		broadcast:    make(chan []byte),
		gameSessions: make(map[string]*game.GameSession),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.registerConnection(conn)

		case conn := <-h.unregister:
			h.unregisterConnection(conn)

		case msg := <-h.inbound:
			// Decide how to handle messages.
			// For example, parse JSON for "action" field and route.
			h.handleInbound(msg)

		case message := <-h.broadcast:
			// Broadcast to all connections if needed
			for conn := range h.connections {
				select {
				case conn.send <- message:
				default:
					// If sending fails, close the connection
					close(conn.send)
					delete(h.connections, conn)
				}
			}
		}
	}
}

func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

func (h *Hub) registerConnection(conn *Connection) {
	h.connections[conn] = true
	fmt.Println("New connection registered!")
}

func (h *Hub) unregisterConnection(conn *Connection) {
	if _, ok := h.connections[conn]; ok {
		delete(h.connections, conn)
		close(conn.send)
		fmt.Println("Connection unregistered!")
	}
}

// handleInbound is where you decode or route the message from a client.
func (h *Hub) handleInbound(msg InboundMessage) {
	// For example, you might parse JSON:
	//   { "type": "JOIN_GAME", "gameID": "123" }
	//   { "type": "MAKE_MOVE", "gameID": "123", "move": "e2e4" }
	// This is just a placeholder to show the structure.

	// Let's do a simple print:
	fmt.Printf("Inbound message from a connection: %s\n", string(msg.Payload))

	// In a real app, you'd unmarshal the JSON, check "type" and "gameID"
	// then do something like:
	//  switch messageType {
	//  case "JOIN_GAME":
	//      h.joinGame(msg.Conn, gameID)
	//  case "MAKE_MOVE":
	//      h.makeMove(gameID, moveData)
	//  ...
	//  }
}

// joinGame is an example of how you'd attach a connection to a game session
// or create a new session if it doesn't exist.
func (h *Hub) joinGame(conn *Connection, gameID string) *game.GameSession {
	session, ok := h.gameSessions[gameID]
	if !ok {
		session = &game.GameSession{
			ID: gameID,
			// Set up initial times, etc.
		}
		h.gameSessions[gameID] = session
	}
	// You could store a reference in conn, like:
	//   conn.gameID = gameID
	// Or keep track in the session itself. Depends on your design.
	return session
}

// makeMove might look up the session, call session.HandleMove, then broadcast new state.
func (h *Hub) makeMove(gameID, move string) error {
	session, ok := h.gameSessions[gameID]
	if !ok {
		return fmt.Errorf("game %s not found", gameID)
	}

	fmt.Println("sessions:", session)
	// session.HandleMove(move)
	// Then broadcast updated state to all players in that game
	return nil
}
