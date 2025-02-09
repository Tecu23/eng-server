package game

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/tecu23/eng-server/pkg/engine"
)

type Manager struct {
	sessions map[uuid.UUID]*GameSession
	mu       sync.RWMutex
}

// NewSimpleManager creates a new manager with in-memory storage
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[uuid.UUID]*GameSession),
	}
}

// CreateSession creates a new game session with the given parameters and registers it.
func (m *Manager) CreateSession(
	conn *websocket.Conn,
	whiteTime, blackTime, whiteIncrement, blackIncremenent int64,
) *GameSession {
	sessionID := uuid.New()

	eng := engine.NewUCIEngine()

	session := &GameSession{
		ID: sessionID,

		Engine: eng,

		Turn: "white",
		FEN:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",

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

	return session
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
