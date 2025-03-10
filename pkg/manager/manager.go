package manager

import (
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/color"
	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/engine"
	"github.com/tecu23/eng-server/pkg/events"
	"github.com/tecu23/eng-server/pkg/game"
	"github.com/tecu23/eng-server/pkg/repository"
)

type Manager struct {
	repository *repository.InMemoryGameRepository
	enginePool *engine.Pool

	publisher *events.Publisher
	logger    *zap.Logger
}

// NewManager creates a new manager with in-memory storage
func NewManager(
	repo *repository.InMemoryGameRepository,
	engPool *engine.Pool,
	logger *zap.Logger,
	publisher *events.Publisher,
) *Manager {
	manager := &Manager{
		repository: repo,
		enginePool: engPool,
		logger:     logger,
		publisher:  publisher,
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
	m.logger.Info("Terminating sessions for connection", zap.String("connection_id", connectionID))

	activeGames, err := m.repository.ListActiveGames()
	if err != nil {
		m.logger.Error(
			"Could not terminate sessions for connection",
			zap.String("connection_id", connectionID),
			zap.Error(err),
		)
	}

	for _, g := range activeGames {
		if g.ConnectionID.String() == connectionID {
			gameID := g.ID
			go func() {
				g.Terminate()
				m.RemoveSession(gameID)
			}()
		}
	}
}

// CreateSession creates a new game session with the given parameters and registers it.
func (m *Manager) CreateSession(
	whiteTime, blackTime, whiteIncrement, blackIncremenent int64,
	turn color.Color,
	fen string,
	connectionId uuid.UUID,
	publisher *events.Publisher,
) (*game.Game, error) {
	sessionID := uuid.New()

	eng, err := m.enginePool.GetEngine()
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

	params := game.CreateGameParams{
		GameID:       sessionID,
		StartPostion: fen,
		TimeControl:  tc,
	}

	session, err := game.CreateGame(params, connectionId, eng, publisher, m.logger)

	if err := m.repository.SaveGame(session); err != nil {
		return nil, err
	}

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
	session, err := m.repository.GetGame(id)
	if err != nil {
		return nil, false
	}
	return session, true
}

// RemoveSession cleans up a finished session
func (m *Manager) RemoveSession(id uuid.UUID) {
	session, err := m.repository.GetGame(id)
	if err != nil {
		m.logger.Error("could not remove game session", zap.Error(err))
		return
	}

	session.Terminate()

	m.logger.Info("removed game session", zap.String("session_id", id.String()))
}
