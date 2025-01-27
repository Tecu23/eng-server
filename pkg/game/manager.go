package game

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tecu23/eng-server/pkg/messages"
)

type GameState struct {
	BoardFEN    string
	WhiteTime   int64
	BlackTime   int64
	CurrentTurn string
	IsCheckmate bool
	IsDraw      bool
}

type Manager interface {
	CreateGameSession(payload messages.StartNewGamePayload) (string, error)
	MakeMove(gameID, move string) (*GameState, error)
}

// SimpleManager is an in-memory implementation
type SimpleManager struct {
	sessions map[string]*GameSession
	mu       sync.Mutex
}

// NewSimpleManager creates a new manager with in-memory storage
func NewSimpleManager() *SimpleManager {
	return &SimpleManager{
		sessions: make(map[string]*GameSession),
	}
}

func (gm *SimpleManager) CreateGameSession(payload messages.StartNewGamePayload) (string, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gameID := fmt.Sprintf("game-%d", len(gm.sessions)+1) // simplistic unique ID
	session := &GameSession{
		GameID:       gameID,
		BoardFEN:     "startpos", // or actual FEN for new game
		WhiteTime:    int64(payload.TimeControl.Initial * 1000),
		BlackTime:    int64(payload.TimeControl.Initial * 1000),
		CurrentTurn:  "white",
		LastMoveTime: GetNow(), // or time.Now()
	}
	gm.sessions[gameID] = session
	return gameID, nil
}

func (gm *SimpleManager) MakeMove(gameID, move string) (*GameState, error) {
	gm.mu.Lock()
	session, ok := gm.sessions[gameID]
	gm.mu.Unlock()
	if !ok {
		return nil, errors.New("game not found")
	}

	if err := session.HandleMove(move); err != nil {
		return nil, err
	}
	return &GameState{
		BoardFEN:    session.BoardFEN,
		WhiteTime:   session.WhiteTime,
		BlackTime:   session.BlackTime,
		CurrentTurn: session.CurrentTurn,
		IsCheckmate: session.IsCheckmate,
		IsDraw:      session.IsDraw,
	}, nil
}

func GetNow() time.Time {
	// this is a hook for test or production
	return time.Now()
}
