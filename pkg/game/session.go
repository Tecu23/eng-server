package game

import (
	"sync"
	"time"
)

type GameSession struct {
	GameID string

	BoardFEN    string
	WhiteTime   int64
	BlackTime   int64
	CurrentTurn string

	IsCheckmate bool
	IsDraw      bool

	LastMoveTime time.Time
	mu           sync.Mutex
}

func (gs *GameSession) HandleMove(_ string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	return nil
}
