// Package game
package game

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/tecu23/eng-server/pkg/engine"
)

type Manager struct {
	sessions map[uuid.UUID]*GameSession
	mu       sync.RWMutex
}

// NewManager creates a new manager with in-memory storage
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[uuid.UUID]*GameSession),
	}
}

// CreateSession creates a new game session with the given parameters and registers it.
func (m *Manager) CreateSession(
	conn *websocket.Conn,
	whiteTime, blackTime, whiteIncrement, blackIncremenent int64,
	turn string,
	fen string,
) (*GameSession, error) {
	sessionID := uuid.New()

	eng, err := engine.NewUCIEngine("./bin/argo_linux_amd64")
	if err != nil {
		return nil, err
	}

	session := &GameSession{
		ID: sessionID,

		Engine: eng,

		Turn: turn,
		FEN:  fen,

		lastMoveTime: time.Now(),

		WhiteTime:      whiteTime,
		BlackTime:      blackTime,
		WhiteIncrement: whiteIncrement,
		BlackIncrement: blackIncremenent,

		Conn: conn,
		done: make(chan bool),
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	// Start sending periodic clock updates?
	go session.startClockTicker()

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
}
