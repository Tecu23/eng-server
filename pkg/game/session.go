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
)

type GameSession struct {
	ID uuid.UUID

	Conn *websocket.Conn

	Engine *engine.UCIEngine

	FEN   string
	Moves []string
	Turn  chess.Color

	Clock *chess.Clock

	done chan bool

	mu      sync.Mutex
	writeMu sync.Mutex

	logger *zap.Logger
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

	// Inform the client about the move and the turn change.
	if s.Conn != nil {
		var payload messages.EngineMovePayload
		var engineMoveMsg messages.OutboundMessage

		payload.Move = bestMove
		payload.Color = turn

		engineMoveMsg.Event = "ENGINE_MOVE"
		engineMoveMsg.Payload = payload

		s.SendJSON(engineMoveMsg)
	}

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
				var payload messages.ClockUpdatePayload
				payload.WhiteTime = tick.White
				payload.BlackTime = tick.Black
				payload.ActiveColor = string(tick.ActiveColor)

				msg := messages.OutboundMessage{
					Event:   "CLOCK_UPDATE",
					Payload: payload,
				}

				if err := s.SendJSON(msg); err != nil {
					s.logger.Error("failed to send clock update", zap.Error(err))
				}

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
				// Handle timeout - notify client that time is up
				var payload messages.TimeupPayload
				payload.Color = string(color)

				msg := messages.OutboundMessage{
					Event:   "TIME_UP",
					Payload: payload,
				}

				if err := s.SendJSON(msg); err != nil {
					s.logger.Error("failed to send timeout notification", zap.Error(err))
				}

				s.logger.Info("player time expired", zap.String("color", string(color)))
			}
		}
	}()
}
