package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tecu23/eng-server/pkg/game"
	"github.com/tecu23/eng-server/pkg/messages"
)

type InboundHubMessage struct {
	Conn    *Connection             // who sent it
	Message messages.InboundMessage // raw JSON or text
}

type Hub struct {
	mu          sync.RWMutex
	connections map[*Connection]bool // Registered connections

	register   chan *Connection       // Incoming registration
	unregister chan *Connection       // Incoming unregistration
	inbound    chan InboundHubMessage // Channel or inbound messages that the hub might route or broadcast

	// broadcast chan []byte // Channel to broadcast to everyone

	gameManager *game.SimpleManager
}

func NewHub(gm *game.SimpleManager) *Hub {
	return &Hub{
		connections: make(map[*Connection]bool),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		inbound:     make(chan InboundHubMessage),
		// broadcast:    make(chan []byte),
		gameManager: gm,
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
			h.handleInbound(msg)
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
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[conn] = true
	fmt.Println("New connection registered!")
}

func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connections[conn]; ok {
		delete(h.connections, conn)
		close(conn.send)
		fmt.Println("Connection unregistered!")
	}
}

// handleInbound is where you decode or route the message from a client.
func (h *Hub) handleInbound(msg InboundHubMessage) {
	switch msg.Message.Type {
	case "START_NEW_GAME":

		var payload messages.StartNewGamePayload
		if err := json.Unmarshal(msg.Message.Payload, &payload); err != nil {
			h.sendError(msg.Conn, "Invalid START_NEW_GAME payload")
			return
		}

		gameID, err := h.gameManager.CreateGameSession(payload)
		if err != nil {
			h.sendError(msg.Conn, err.Error())
			return
		}

		resp := messages.OutboundMessage{
			Type: "GAME_CREATED",
			Payload: messages.GameCreatedPayload{
				GameID:      gameID,
				InitialFEN:  "startpos",
				WhiteTime:   30000,
				BlackTime:   30000,
				CurrentTurn: "white",
			},
		}

		msg.Conn.SendJSON(resp)
	case "MAKE_MOVE":
		var payload messages.MakeMovePayload
		if err := json.Unmarshal(msg.Message.Payload, &payload); err != nil {
			h.sendError(msg.Conn, "Invalid MAKE_MOVE payload")
			return
		}
		state, err := h.gameManager.MakeMove(payload.GameID, payload.Move)
		if err != nil {
			h.sendError(msg.Conn, err.Error())
			return
		}
		// Broadcast or just send to this connection:
		resp := messages.OutboundMessage{
			Type: "GAME_STATE",
			Payload: messages.GameStatePayload{
				GameID:      payload.GameID,
				BoardFEN:    state.BoardFEN,
				WhiteTime:   state.WhiteTime,
				BlackTime:   state.BlackTime,
				CurrentTurn: state.CurrentTurn,
				IsCheckmate: state.IsCheckmate,
				IsDraw:      state.IsDraw,
			},
		}
		msg.Conn.SendJSON(resp)
	default:
		h.sendError(msg.Conn, "Unknown message type")
	}
}

func (h *Hub) sendError(conn *Connection, msg string) {
	resp := messages.OutboundMessage{
		Type: "ERROR",
		Payload: messages.ErrorPayload{
			Message: msg,
		},
	}
	conn.SendJSON(resp)
}
