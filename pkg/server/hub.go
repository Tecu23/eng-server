package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/chess"
	"github.com/tecu23/eng-server/pkg/game"
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

	logger *zap.Logger
}

// NewHub creates a new hub
func NewHub(gm *game.Manager, logger *zap.Logger) *Hub {
	return &Hub{
		connections: make(map[*Connection]bool),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		inbound:     make(chan InboundHubMessage),
		broadcast:   make(chan []byte),
		gameManager: gm,
		logger:      logger,
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

// Register should
func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

func (h *Hub) registerConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[conn] = true
	h.logger.Info("New connection registered", zap.Int("total_connections", len(h.connections)))

	var payload messages.ConnectedPayload
	payload.ConnectionId = conn.ID.String()

	msg := messages.OutboundMessage{
		Event:   "CONNECTED",
		Payload: payload,
	}

	h.sendMessage(conn, msg)
}

// Unregister should
func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

func (h *Hub) unregisterConnection(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connections[conn]; ok {
		delete(h.connections, conn)
		close(conn.send)
		h.logger.Info("Connection unregistered", zap.Int("total_connections", len(h.connections)))

	}
}

// handleInbound is where the message from a client is decoded and handled
func (h *Hub) handleInbound(msg InboundHubMessage) {
	switch msg.Message.Event {
	case "CREATE_SESSION":
		var payload messages.CreateSession
		if err := json.Unmarshal(msg.Message.Payload, &payload); err != nil {
			h.logger.Error("Invalid CREATE_SESSION payload", zap.Error(err))
			h.sendError(msg.Conn, "Invalid START_NEW_GAME payload")
			return
		}

		var clr chess.Color

		if payload.Color == "w" {
			clr = chess.White
		} else {
			clr = chess.Black
		}

		gameSession, err := h.gameManager.CreateSession(
			msg.Conn.ws,
			payload.TimeControl.WhiteTime,
			payload.TimeControl.BlackTime,
			payload.TimeControl.WhiteIncrement,
			payload.TimeControl.BlackIncrement,
			clr,
			payload.InitialFen,
		)
		if err != nil {
			h.logger.Error("Error creating game session", zap.Error(err))
			h.sendError(msg.Conn, err.Error())
			return
		}

		resp := messages.OutboundMessage{
			Event: "GAME_CREATED",
			Payload: messages.GameCreatedPayload{
				GameID:      gameSession.ID.String(),
				InitialFEN:  gameSession.FEN,
				WhiteTime:   gameSession.Clock.GetRemainingTime().White,
				BlackTime:   gameSession.Clock.GetRemainingTime().Black,
				CurrentTurn: gameSession.Turn,
			},
		}

		h.logger.Info("Game session created", zap.String("game_id", gameSession.ID.String()))
		h.sendMessage(msg.Conn, resp)

	case "MAKE_MOVE":
		var payload messages.MakeMovePayload
		if err := json.Unmarshal(msg.Message.Payload, &payload); err != nil {
			h.logger.Error("Invalid MAKE_MOVE payload", zap.Error(err))
			h.sendError(msg.Conn, "Invalid MAKE_MOVE payload")
			return
		}

		id, err := uuid.Parse(payload.GameID)
		if err != nil {
			h.logger.Error("Could not parse game session id", zap.Error(err))
			h.sendError(msg.Conn, err.Error())
			return
		}

		session, ok := h.gameManager.GetSession(id)
		if !ok {
			h.logger.Error("Could not find session", zap.Error(err))
			h.sendError(
				msg.Conn,
				fmt.Sprintf("Could not find session with session id %s", payload.GameID),
			)
			return
		}

		err = session.ProcessMove(payload.Move)
		if err != nil {
			h.logger.Error("Could not process move", zap.Error(err))
			h.sendError(msg.Conn, err.Error())
			return
		}

		resp := messages.OutboundMessage{
			Event: "GAME_STATE",
			Payload: messages.GameStatePayload{
				GameID:      session.ID.String(),
				BoardFEN:    session.FEN,
				WhiteTime:   session.Clock.GetRemainingTime().White,
				BlackTime:   session.Clock.GetRemainingTime().Black,
				CurrentTurn: session.Turn,
			},
		}

		h.logger.Info("Game state updated", zap.String("game_id", session.ID.String()))
		h.sendMessage(msg.Conn, resp)

		// Call engine to make an engine move as well
		session.ProcessEngineMove()

	default:
		h.logger.Warn("Unknown message type", zap.String("event", msg.Message.Event))
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
