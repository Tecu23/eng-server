// Package game
package game

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/pkg/chess"
	"github.com/tecu23/eng-server/pkg/engine"
)

type Manager struct {
	sessions map[uuid.UUID]*GameSession
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewManager creates a new manager with in-memory storage
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		sessions: make(map[uuid.UUID]*GameSession),
		logger:   logger,
	}
}

// CreateSession creates a new game session with the given parameters and registers it.
func (m *Manager) CreateSession(
	conn *websocket.Conn,
	whiteTime, blackTime, whiteIncrement, blackIncremenent int64,
	turn chess.Color,
	fen string,
) (*GameSession, error) {
	sessionID := uuid.New()

	eng, err := engine.NewUCIEngine("./bin/argo_linux_amd64")
	if err != nil {
		m.logger.Error("failed to initialize engine", zap.Error(err))
		return nil, err
	}

	tc := chess.TimeControl{
		WhiteTime:       whiteTime,
		WhiteIncrement:  whiteIncrement,
		BlackTime:       blackTime,
		BlackIncrement:  blackIncremenent,
		MovesPerControl: 40,
		TimingMethod:    chess.IncrementTiming,
	}

	clock := chess.NewClock(tc)

	session := &GameSession{
		ID: sessionID,

		Engine: eng,

		Turn:  turn,
		FEN:   fen,
		Clock: clock,

		Conn:   conn,
		done:   make(chan bool),
		logger: m.logger,
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	m.logger.Info("created new game session", zap.String("session_id", sessionID.String()))

	// Start sending periodic clock updates
	go session.Clock.Start()
	go session.StartClockUpdates()
	go session.StartTimeoutMonitor()

	return session, nil
}

// GetSession returns a session by ID
func (m *Manager) GetSession(id uuid.UUID) (*GameSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// RemoveSession cleans up a finished session
func (m *Manager) RemoveSession(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	m.logger.Info("removed game session", zap.String("session_id", id.String()))
}
