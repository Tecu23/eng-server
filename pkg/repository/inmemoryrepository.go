package repository

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/pkg/game"
)

// InMemoryGameRepository in an in-memory implementation of GameRepository
type InMemoryGameRepository struct {
	games  map[uuid.UUID]*game.Game
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewInMemoryRepository creates a new in-memory repository
func NewInMemoryRepository(logger *zap.Logger) *InMemoryGameRepository {
	return &InMemoryGameRepository{
		games:  make(map[uuid.UUID]*game.Game),
		logger: logger,
	}
}

// SaveGame saves a game to the repository
func (r *InMemoryGameRepository) SaveGame(game *game.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.games[game.ID] = game
	return nil
}

// GetGame retrieves a game by ID
func (r *InMemoryGameRepository) GetGame(id uuid.UUID) (*game.Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	game, ok := r.games[id]
	if !ok {
		return nil, errors.New("game not found")
	}

	return game, nil
}

// ListActiveGames returns all active games
func (r *InMemoryGameRepository) ListActiveGames() ([]*game.Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var activeGames []*game.Game
	for _, g := range r.games {
		if g.Status == game.StatusActive {
			activeGames = append(activeGames, g)
		}
	}

	return activeGames, nil
}
