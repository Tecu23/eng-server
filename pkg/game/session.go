package game

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/tecu23/eng-server/pkg/engine"
	"github.com/tecu23/eng-server/pkg/messages"
)

type GameSession struct {
	ID uuid.UUID

	Engine *engine.UCIEngine

	Turn string

	Moves []string

	FEN string

	WhiteTime      int64
	BlackTime      int64
	WhiteIncrement int64
	BlackIncrement int64

	lastMoveTime time.Time

	Conn *websocket.Conn

	done chan bool

	mu sync.Mutex
}

func (s *GameSession) startClockTicker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.updateClock()
		case <-s.done:
			return
		}
	}
}

func (s *GameSession) updateClock() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate elapsed time since the last move.
	elapsed := time.Since(s.lastMoveTime).Milliseconds()
	var remainingTime int64

	// Determine which clock to update based on whose turn it is.
	if s.Turn == "white" {
		remainingTime = s.WhiteTime - elapsed
	} else {
		remainingTime = s.BlackTime - elapsed
	}

	// Send a clock update to the client.
	if s.Conn != nil {
		var payload messages.TimeUpdatePayload
		payload.Remaining = remainingTime
		payload.Color = s.Turn

		updateMsg := map[string]interface{}{
			"event":   "CLOCK_UPDATE",
			"payload": payload,
		}
		s.Conn.WriteJSON(updateMsg)
	}

	// If the clock reaches zero, handle the timeout.
	if remainingTime <= 0 {
		s.handleTimeout()
	}
}

func (s *GameSession) handleTimeout() {
	// Signal to stop the clock ticker.
	select {
	case s.done <- true:
	default:
	}

	var result string
	if s.Turn == "w" {
		result = "Black wins on time"
	} else {
		result = "White wins on time"
	}

	if s.Conn != nil {
		s.Conn.WriteJSON(map[string]interface{}{
			"event":  "GAME_OVER",
			"reason": result,
		})
	}

	// Optionally shut down the engine if one was running.
	if s.Engine != nil {
		s.Engine.Close()
	}
}

func (s *GameSession) ProcessMove(move string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate how much time has elapsed since the turn started.
	elapsed := time.Since(s.lastMoveTime).Milliseconds()

	// Update the clock for the active player.
	if s.Turn == "w" {
		s.WhiteTime = s.WhiteTime - elapsed + s.WhiteIncrement
		s.Turn = "b" // Switch turn.
	} else {
		s.BlackTime = s.BlackTime - elapsed + s.BlackIncrement
		s.Turn = "w"
	}

	// Record the move.
	s.Moves = append(s.Moves, move)

	// Reset the timer for the new move.
	s.lastMoveTime = time.Now()

	return nil
}

func (s *GameSession) ProcessEngineMove() {
	s.mu.Lock()
	blackTime := s.BlackTime
	whiteTime := s.WhiteTime
	movestogo := len(s.Moves) / 2
	moves := s.Moves
	fen := s.FEN
	conn := s.Conn
	turn := s.Turn
	s.mu.Unlock()

	command := fmt.Sprintf("position fen %s moves %s", fen, strings.Join(moves, " "))
	fmt.Println(command)
	if err := s.Engine.SendCommand(command); err != nil {
		// Handle error
		return
	}

	command = fmt.Sprintf("go wtime %d btime %d movestogo %d", whiteTime, blackTime, 40-movestogo)
	fmt.Println(command, movestogo)
	if err := s.Engine.SendCommand(command); err != nil {
		// Handle error
		return
	}

	// Wait for the best move from the engine.
	bestMove := <-s.Engine.BestMoveChan

	// Process the move as if the engine made it.
	s.ProcessMove(bestMove)

	// Inform the client about the move and the turn change.
	if conn != nil {
		var payload messages.EngineMovePayload
		payload.Move = bestMove
		payload.Color = turn

		conn.WriteJSON(map[string]interface{}{
			"event":   "ENGINE_MOVE",
			"payload": payload,
		})
	}
}
