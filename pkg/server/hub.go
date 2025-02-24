package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/tecu23/eng-server/pkg/game"
	"github.com/tecu23/eng-server/pkg/messages"
)

// InboundHubMessage are the messages that the hub receives
type InboundHubMessage struct {
	Conn    *Connection             // who sent it
	Message messages.InboundMessage // raw JSON or texthub
}

// Hub should keep track of all active connection. Also be responsible of registering/unregistering connections
// Messages come from the inbound channel and are redirected to the corrected game session or broadcast
type Hub struct {
	mu          sync.RWMutex         // Mutex to protect direct access to the connections map.
	connections map[*Connection]bool // Registered connections

	register   chan *Connection       // Incoming registration
	unregister chan *Connection       // Incoming unregistration
	inbound    chan InboundHubMessage // Channel or inbound messages that the hub might route or broadcast

	broadcast chan []byte // Channel to broadcast to everyone

	gameManager *game.Manager
}

// NewHub creates a new hub
func NewHub(gm *game.Manager) *Hub {
	return &Hub{
		connections: make(map[*Connection]bool),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		inbound:     make(chan InboundHubMessage),
		broadcast:   make(chan []byte),
		gameManager: gm,
	}
}

// Run is the main execution of the hub
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
	fmt.Println("New connection registered!", len(h.connections))

	var payload messages.ConnectedPayload
	payload.ConnectionId = conn.ID.String()

	msg := messages.OutboundMessage{
		Event:   "CONNECTED",
		Payload: payload,
	}

	h.sendMessage(conn, msg)
}

func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connections[conn]; ok {
		delete(h.connections, conn)
		close(conn.send)
		fmt.Println("Connection unregistered!", len(h.connections))
	}
}

// handleInbound is where you decode or route the message from a client.
func (h *Hub) handleInbound(msg InboundHubMessage) {
	switch msg.Message.Event {
	case "CREATE_SESSION":
		var payload messages.CreateSession
		if err := json.Unmarshal(msg.Message.Payload, &payload); err != nil {
			h.sendError(msg.Conn, "Invalid START_NEW_GAME payload")
			return
		}
		fmt.Println(payload)

		gameSession, err := h.gameManager.CreateSession(
			msg.Conn.ws,
			payload.TimeControl.WhiteTime,
			payload.TimeControl.BlackTime,
			payload.TimeControl.WhiteIncrement,
			payload.TimeControl.BlackIncrement,
			payload.Color,
			payload.InitialFen,
		)
		if err != nil {
			h.sendError(msg.Conn, err.Error())
			return
		}

		resp := messages.OutboundMessage{
			Event: "GAME_CREATED",
			Payload: messages.GameCreatedPayload{
				GameID:      gameSession.ID.String(),
				InitialFEN:  gameSession.FEN,
				WhiteTime:   gameSession.WhiteTime,
				BlackTime:   gameSession.BlackTime,
				CurrentTurn: gameSession.Turn,
			},
		}

		msg.Conn.SendJSON(resp)
	case "MAKE_MOVE":
		var payload messages.MakeMovePayload
		if err := json.Unmarshal(msg.Message.Payload, &payload); err != nil {
			h.sendError(msg.Conn, "Invalid MAKE_MOVE payload")
			return
		}

		id, err := uuid.Parse(payload.GameID)
		if err != nil {
			h.sendError(msg.Conn, err.Error())
			return
		}

		session, ok := h.gameManager.GetSession(id)
		if !ok {
			h.sendError(
				msg.Conn,
				fmt.Sprintf("Could not find session with session id %s", payload.GameID),
			)
			return
		}

		err = session.ProcessMove(payload.Move)
		if err != nil {
			h.sendError(msg.Conn, err.Error())
			return
		}

		resp := messages.OutboundMessage{
			Event: "GAME_STATE",
			Payload: messages.GameStatePayload{
				GameID:      session.ID.String(),
				BoardFEN:    session.FEN,
				WhiteTime:   session.WhiteTime,
				BlackTime:   session.BlackTime,
				CurrentTurn: session.Turn,
			},
		}

		msg.Conn.SendJSON(resp)

		// Call engine to make an engine move as well
		session.ProcessEngineMove()

	default:
		h.sendError(msg.Conn, "Unknown message type")
	}
}

func (h *Hub) sendError(conn *Connection, msg string) {
	resp := messages.OutboundMessage{
		Event: "ERROR",
		Payload: messages.ErrorPayload{
			Message: msg,
		},
	}
	h.sendMessage(conn, resp)
}

func (h *Hub) sendMessage(conn *Connection, msg messages.OutboundMessage) {
	conn.SendJSON(msg)
}
