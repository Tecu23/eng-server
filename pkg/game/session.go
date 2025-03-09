package game

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/chess"
	"github.com/tecu23/eng-server/pkg/engine"
	"github.com/tecu23/eng-server/pkg/events"
)

type GameSession struct {
	ID     uuid.UUID
	Conn   *websocket.Conn
	Engine *engine.UCIEngine

	FEN   string
	Moves []string
	Turn  chess.Color
	Clock *chess.Clock

	done chan bool

	mu      sync.Mutex
	writeMu sync.Mutex

	publisher *events.Publisher
	logger    *zap.Logger
}

func (s *GameSession) ProcessMove(move string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Record the move.
	s.Moves = append(s.Moves, move)
	s.Turn = s.Turn.Opp()
	s.Clock.Switch()

	s.logger.Info(
		"processed move",
		zap.String("move", move),
		zap.String("new_turn", string(s.Turn)),
	)

	// Publish move processed event
	s.publisher.Publish(events.Event{
		Type:   events.EventMoveProcessed,
		GameID: s.ID.String(),
		Payload: messages.GameStatePayload{
			GameID:      s.ID.String(),
			BoardFEN:    s.FEN,
			WhiteTime:   s.Clock.GetRemainingTime().White,
			BlackTime:   s.Clock.GetRemainingTime().Black,
			CurrentTurn: s.Turn,
		},
	})

	return nil
}

func (s *GameSession) ProcessEngineMove() {
	s.mu.Lock()
	wTime, bTime, mvs, fen, turn := s.Clock.GetRemainingTime().White, s.Clock.GetRemainingTime().Black, s.Moves, s.FEN, s.Turn
	s.mu.Unlock()

	command := fmt.Sprintf("position fen %s moves %s", fen, strings.Join(mvs, " "))
	if err := s.Engine.SendCommand(command); err != nil {
		// Handle error
		s.logger.Error("engine command error", zap.Error(err))
		return
	}

	movestogo := len(mvs) / 2

	command = fmt.Sprintf(
		"go wtime %d btime %d movestogo %d",
		wTime,
		bTime,
		40-movestogo,
	)
	if err := s.Engine.SendCommand(command); err != nil {
		// Handle error
		s.logger.Error("engine command error", zap.Error(err))

		return
	}

	// Wait for the best move from the engine.
	bestMove := <-s.Engine.BestMoveChan

	// Process the move as if the engine made it.
	if err := s.ProcessMove(bestMove); err != nil {
		s.logger.Error("failed to process engine move", zap.Error(err))
		return
	}

	// Publish engine moved event
	s.publisher.Publish(events.Event{
		Type:   events.EventEngineMoved,
		GameID: s.ID.String(),
		Payload: messages.EngineMovePayload{
			Move:  bestMove,
			Color: turn,
		},
	})

	s.logger.Info("engine move processed", zap.String("move", bestMove))
}

func (s *GameSession) SendJSON(msg messages.OutboundMessage) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.Conn.WriteJSON(msg)
}

func (s *GameSession) StartClockUpdates() {
	go func() {
		tickChan := s.Clock.GetTickChannel()
		for {
			select {
			case <-s.done:
				return
			case tick := <-tickChan:
				// Publish clock update event
				s.publisher.Publish(events.Event{
					Type:   events.EventClockUpdated,
					GameID: s.ID.String(),
					Payload: messages.ClockUpdatePayload{
						WhiteTime:   tick.White,
						BlackTime:   tick.Black,
						ActiveColor: string(tick.ActiveColor),
					},
				})
			}
		}
	}()
}

func (s *GameSession) StartTimeoutMonitor() {
	go func() {
		timeupChan := s.Clock.GetTimeupChannel()
		for {
			select {
			case <-s.done:
				return
			case color := <-timeupChan:
				// Publish time up event
				s.publisher.Publish(events.Event{
					Type:   events.EventTimeUp,
					GameID: s.ID.String(),
					Payload: messages.TimeupPayload{
						Color: string(color),
					},
				})
				s.logger.Info("player time expired", zap.String("color", string(color)))
			}
		}
	}()
}

func (s *GameSession) Terminate() {
	close(s.done)
	s.Engine.Close()

	// Publish game terminated event
	s.publisher.Publish(events.Event{
		Type:   events.EventGameTerminated,
		GameID: s.ID.String(),
		Payload: map[string]string{
			"game_id": s.ID.String(),
		},
	})
}
