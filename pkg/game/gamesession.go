package game

import (
	"sync"
	"time"
)

type GameSession struct {
	ID                 string
	WhiteTimeRemaining int64 // milliseconds
	BlackTimeRemaining int64
	CurrentPlayer      string // "white" or "black"
	LastMoveTime       time.Time
	// Add more as needed (board state, move history, etc.)

	mu sync.Mutex
}

// UpdateTime is a simple placeholder for updating clocks.
func (gs *GameSession) UpdateTime() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(gs.LastMoveTime).Milliseconds()
	if gs.CurrentPlayer == "white" {
		gs.WhiteTimeRemaining -= elapsed
	} else {
		gs.BlackTimeRemaining -= elapsed
	}
	gs.LastMoveTime = now
}
