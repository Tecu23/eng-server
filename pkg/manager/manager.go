package manager

import (
	"sync"

	"github.com/corentings/chess/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/color"
	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/engine"
	"github.com/tecu23/eng-server/pkg/events"
	"github.com/tecu23/eng-server/pkg/game"
)

type Manager struct {
	sessions  map[uuid.UUID]*game.Game
	mu        sync.RWMutex
	publisher *events.Publisher
	logger    *zap.Logger
}

// NewManager creates a new manager with in-memory storage
func NewManager(logger *zap.Logger, publisher *events.Publisher) *Manager {
	manager := &Manager{
		sessions:  make(map[uuid.UUID]*game.Game),
		logger:    logger,
		publisher: publisher,
	}

	// Set up event handlers
	manager.setupEventHandlers()

	return manager
}

// setupEventHandlers sets up event handlers for the game manager
func (m *Manager) setupEventHandlers() {
	// Handle connection closed events
	m.publisher.Subscribe(events.EventConnectionClosed, func(event events.Event) {
		payload, ok := event.Payload.(map[string]string)
		if !ok {
			m.logger.Error("Invalid connection closed payload type")
			return
		}

		connectionID := payload["connection_id"]

		// Find all game sessions associated with this connection and terminate them
		m.terminateSessionsByConnectionID(connectionID)
	})

	// Handle game terminated events
	m.publisher.Subscribe(events.EventGameTerminated, func(event events.Event) {
		// Remove the session from the manager
		if event.GameID != "" {
			gameID, err := uuid.Parse(event.GameID)
			if err != nil {
				m.logger.Error("Invalid game ID in game terminated event", zap.Error(err))
				return
			}
			m.RemoveSession(gameID)
		}
	})
}

// terminateSessionsByConnectionID finds and terminates all game sessions for a connection
func (m *Manager) terminateSessionsByConnectionID(connectionID string) {
	// This is a placeholder - you would need to implement a way to track
	// which sessions are associated with which connections
	m.logger.Info("Terminating sessions for connection", zap.String("connection_id", connectionID))

	// Example implementation:
	// m.mu.RLock()
	// for id, session := range m.sessions {
	//     if session.ConnectionID == connectionID {
	//         // Make a copy of the ID to avoid issues with the defer and loop variable
	//         sessionID := id
	//         go func() {
	//             session.Terminate()
	//             m.RemoveSession(sessionID)
	//         }()
	//     }
	// }
	// m.mu.RUnlock()
}

// CreateSession creates a new game session with the given parameters and registers it.
func (m *Manager) CreateSession(
	conn *websocket.Conn,
	whiteTime, blackTime, whiteIncrement, blackIncremenent int64,
	turn color.Color,
	fen string,
	publisher *events.Publisher,
) (*game.Game, error) {
	sessionID := uuid.New()

	eng, err := engine.NewUCIEngine("./bin/argo_linux_amd64", m.logger)
	if err != nil {
		m.logger.Error("failed to initialize engine", zap.Error(err))
		return nil, err
	}

	tc := game.TimeControl{
		WhiteTime:       whiteTime,
		WhiteIncrement:  whiteIncrement,
		BlackTime:       blackTime,
		BlackIncrement:  blackIncremenent,
		MovesPerControl: 40,
		TimingMethod:    game.IncrementTiming,
	}

	clock := game.NewClock(tc)

	var internalGame *chess.Game

	if fen == "" || fen == "startpos" {
		internalGame = chess.NewGame()
	} else {
		internalGame = chess.NewGame()
	}

	session := &game.Game{
		ID: sessionID,

		Engine: eng,

		Game:   internalGame,
		Clock:  clock,
		Status: game.StatusPending,

		Conn:      conn,
		Done:      make(chan bool),
		Logger:    m.logger,
		Publisher: publisher,
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	m.logger.Info("created new game session", zap.String("session_id", sessionID.String()))

	// Start sending periodic clock updates
	go session.Clock.Start()
	go session.StartClockUpdates()
	go session.StartTimeoutMonitor()

	// Publish game created event
	publisher.Publish(events.Event{
		Type:   events.EventGameCreated,
		GameID: sessionID.String(),
		Payload: messages.GameCreatedPayload{
			GameID:      sessionID.String(),
			InitialFEN:  fen,
			WhiteTime:   whiteTime,
			BlackTime:   blackTime,
			CurrentTurn: turn,
		},
	})

	return session, nil
}

// GetSession returns a session by ID
func (m *Manager) GetSession(id uuid.UUID) (*game.Game, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// RemoveSession cleans up a finished session
func (m *Manager) RemoveSession(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		// Ensure we close the engine and channels
		session.Engine.Close()
		close(session.Done)
	}

	m.logger.Info("removed game session", zap.String("session_id", id.String()))
}
