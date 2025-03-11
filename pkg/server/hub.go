package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/color"
	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/events"
	"github.com/tecu23/eng-server/pkg/manager"
)

// InboundHubMessage are the messages that the hub receives
type InboundHubMessage struct {
	Conn    *Connection             // who sent it
	Message messages.InboundMessage // raw JSON or texthub
}

// Hub should keep track of all active connection. Also be responsible of registering/unregistering connections
// Messages come from the inbound channel and are redirected to the corrected game session or broadcast
type Hub struct {
	mu sync.RWMutex // Mutex to protect direct access to the connections map.

	connections     map[*Connection]bool     // Registered connections
	gameConnections map[string]*Connection   // Maps game IDs to connections
	connGames       map[*Connection][]string // Maps connections to their game IDs

	register   chan *Connection       // Incoming registration
	unregister chan *Connection       // Incoming unregistration
	inbound    chan InboundHubMessage // Channel or inbound messages that the hub might route or broadcast

	broadcast chan []byte // Channel to broadcast to everyone

	gameManager *manager.Manager
	publisher   *events.Publisher

	logger *zap.Logger
}

// NewHub creates a new hub
func NewHub(gm *manager.Manager, publisher *events.Publisher, logger *zap.Logger) *Hub {
	hub := &Hub{
		connections:     make(map[*Connection]bool),
		gameConnections: make(map[string]*Connection),
		connGames:       make(map[*Connection][]string),
		register:        make(chan *Connection),
		unregister:      make(chan *Connection),
		inbound:         make(chan InboundHubMessage),
		broadcast:       make(chan []byte),
		gameManager:     gm,
		publisher:       publisher,
		logger:          logger,
	}

	// Subscribe to events
	hub.setupEventHandlers()

	return hub
}

// setupEventHandlers sets up the hub's event handlers
func (h *Hub) setupEventHandlers() {
	// Handle game created events
	h.publisher.Subscribe(events.EventGameCreated, func(event events.Event) {
		payload, ok := event.Payload.(messages.GameCreatedPayload)
		if !ok {
			h.logger.Error("Invalid game created payload type")
			return
		}
		// Find the connection associated with this game
		// This mapping would need to be maintained separately
		conn := h.findConnectionForGame(event.GameID)
		if conn == nil {
			h.logger.Error(
				"Could not find connection for game",
				zap.String("game_id", event.GameID),
			)
			return
		}

		resp := messages.OutboundMessage{
			Event:   "GAME_CREATED",
			Payload: payload,
		}

		h.sendMessage(conn, resp)
	})

	// Handle engine move events
	h.publisher.Subscribe(events.EventEngineMoved, func(event events.Event) {
		payload, ok := event.Payload.(messages.EngineMovePayload)
		if !ok {
			h.logger.Error("Invalid engine move payload type")
			return
		}

		conn := h.findConnectionForGame(event.GameID)
		if conn == nil {
			h.logger.Error(
				"Could not find connection for game",
				zap.String("game_id", event.GameID),
			)
			return
		}

		resp := messages.OutboundMessage{
			Event:   "ENGINE_MOVE",
			Payload: payload,
		}

		h.sendMessage(conn, resp)
	})

	// Handle clock update events
	h.publisher.Subscribe(events.EventClockUpdated, func(event events.Event) {
		payload, ok := event.Payload.(messages.ClockUpdatePayload)
		if !ok {
			h.logger.Error("Invalid clock update payload type")
			return
		}

		conn := h.findConnectionForGame(event.GameID)
		if conn == nil {
			h.logger.Error(
				"Could not find connection for game",
				zap.String("game_id", event.GameID),
			)
			return
		}

		resp := messages.OutboundMessage{
			Event:   "CLOCK_UPDATE",
			Payload: payload,
		}

		h.sendMessage(conn, resp)
	})

	// Handle time up events
	h.publisher.Subscribe(events.EventTimeUp, func(event events.Event) {
		payload, ok := event.Payload.(messages.TimeupPayload)
		if !ok {
			h.logger.Error("Invalid time up payload type")
			return
		}

		conn := h.findConnectionForGame(event.GameID)
		if conn == nil {
			h.logger.Error(
				"Could not find connection for game",
				zap.String("game_id", event.GameID),
			)
			return
		}

		resp := messages.OutboundMessage{
			Event:   "TIME_UP",
			Payload: payload,
		}

		h.sendMessage(conn, resp)
	})
}

// findConnectionForGame finds the connection associated with a game
func (h *Hub) findConnectionForGame(gameID string) *Connection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conn, exists := h.gameConnections[gameID]
	if !exists {
		return nil
	}
	return conn
}

// associateConnectionWithGame registers a connection as the owner of a game
func (h *Hub) associateConnectionWithGame(conn *Connection, gameID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to game->connection mapping
	h.gameConnections[gameID] = conn

	// Add to connection->games mapping
	h.connGames[conn] = append(h.connGames[conn], gameID)

	h.logger.Info("Associated connection with game",
		zap.String("connection_id", conn.ID.String()),
		zap.String("game_id", gameID))
}

// removeGameAssociations removes all game associations for a connection
func (h *Hub) removeGameAssociations(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get all games for this connection
	games, exists := h.connGames[conn]
	if !exists {
		return
	}

	// Remove each game->connection mapping
	for _, gameID := range games {
		delete(h.gameConnections, gameID)
		h.logger.Info("Removed game association",
			zap.String("game_id", gameID),
			zap.String("connection_id", conn.ID.String()))
	}

	// Remove the connection->games mapping
	delete(h.connGames, conn)
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
	// First, remove any game associations
	h.removeGameAssociations(conn)

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.connections[conn]; ok {
		delete(h.connections, conn)
		close(conn.send)
		h.logger.Info("Connection unregistered", zap.Int("total_connections", len(h.connections)))

		// Publish connection closed event
		h.publisher.Publish(events.Event{
			Type: events.EventConnectionClosed,
			Payload: map[string]string{
				"connection_id": conn.ID.String(),
			},
		})

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

		var clr color.Color

		if payload.Color == "w" {
			clr = color.White
		} else {
			clr = color.Black
		}

		gameSession, err := h.gameManager.CreateSession(
			payload.TimeControl.WhiteTime,
			payload.TimeControl.BlackTime,
			payload.TimeControl.WhiteIncrement,
			payload.TimeControl.BlackIncrement,
			clr,
			payload.InitialFen,
			msg.Conn.ID,
			h.publisher,
		)
		if err != nil {
			h.logger.Error("Error creating game session", zap.Error(err))
			h.sendError(msg.Conn, err.Error())
			return
		}

		// Associate the connection with the game ID
		h.associateConnectionWithGame(msg.Conn, gameSession.ID.String())

		h.logger.Info("Game session created", zap.String("game_id", gameSession.ID.String()))

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

func (h *Hub) Shutdown() error {
	return nil
}
