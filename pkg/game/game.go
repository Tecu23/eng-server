package game

import (
	"fmt"
	"sync"

	"github.com/corentings/chess/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tecu23/eng-server/internal/color"
	"github.com/tecu23/eng-server/internal/messages"
	"github.com/tecu23/eng-server/pkg/engine"
	"github.com/tecu23/eng-server/pkg/events"
)

type CreateGameParams struct {
	GameID       uuid.UUID
	StartPostion string
	TimeControl  TimeControl
}

type GameStatus string

const (
	StatusActive    GameStatus = "active"
	StatusPending   GameStatus = "pending"
	StatusCompleted GameStatus = "completed"
)

type Game struct {
	ID     uuid.UUID
	Engine *engine.UCIEngine

	ConnectionID uuid.UUID

	Clock  *Clock
	Game   *chess.Game
	Status GameStatus

	done chan bool

	mu sync.Mutex

	Publisher *events.Publisher
	Logger    *zap.Logger
}

func CreateGame(
	params CreateGameParams,
	connectionId uuid.UUID,
	eng *engine.UCIEngine,
	publisher *events.Publisher,
	logger *zap.Logger,
) (*Game, error) {
	clock := NewClock(params.TimeControl)

	var internalGame *chess.Game

	if params.StartPostion == "" || params.StartPostion == "startpos" {
		internalGame = chess.NewGame()
	} else {
		internalGame = chess.NewGame()
	}

	session := &Game{
		ID: params.GameID,

		ConnectionID: connectionId,

		Engine: eng,

		Game:   internalGame,
		Clock:  clock,
		Status: StatusPending,

		done:      make(chan bool),
		Logger:    logger,
		Publisher: publisher,
	}

	return session, nil
}

func (s *Game) ProcessMove(move string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Record the move.
	s.Clock.Switch()
	s.Game.PushMove(move, nil)

	s.Logger.Info(
		"processed move",
		zap.String("move", move),
		zap.String("new_turn", string(s.Game.Position().Turn())),
	)

	// Publish move processed event
	s.Publisher.Publish(events.Event{
		Type:   events.EventMoveProcessed,
		GameID: s.ID.String(),
		Payload: messages.GameStatePayload{
			GameID:    s.ID.String(),
			WhiteTime: s.Clock.GetRemainingTime().White,
			BlackTime: s.Clock.GetRemainingTime().Black,
		},
	})

	return nil
}

func (s *Game) ProcessEngineMove() {
	s.mu.Lock()
	wTime, bTime, mvs, fen, turn := s.Clock.GetRemainingTime().White, s.Clock.GetRemainingTime().Black, s.Game.Moves(), s.Game.FEN(), s.Game.Position().
		Turn()
	s.mu.Unlock()

	command := fmt.Sprintf("position fen %s", fen)
	if err := s.Engine.SendCommand(command); err != nil {
		// Handle error
		s.Logger.Error("engine command error", zap.Error(err))
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
		s.Logger.Error("engine command error", zap.Error(err))

		return
	}

	// Wait for the best move from the engine.
	bestMove := <-s.Engine.BestMoveChan

	// Process the move as if the engine made it.
	if err := s.ProcessMove(bestMove); err != nil {
		s.Logger.Error("failed to process engine move", zap.Error(err))
		return
	}

	// Publish engine moved event
	s.Publisher.Publish(events.Event{
		Type:   events.EventEngineMoved,
		GameID: s.ID.String(),
		Payload: messages.EngineMovePayload{
			Move:  bestMove,
			Color: color.Color(turn),
		},
	})

	s.Logger.Info("engine move processed", zap.String("move", bestMove))
}

func (s *Game) StartClockUpdates() {
	go func() {
		tickChan := s.Clock.GetTickChannel()
		for {
			select {
			case <-s.done:
				return
			case tick := <-tickChan:
				// Publish clock update event
				s.Publisher.Publish(events.Event{
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

func (s *Game) StartTimeoutMonitor() {
	go func() {
		timeupChan := s.Clock.GetTimeupChannel()
		for {
			select {
			case <-s.done:
				return
			case color := <-timeupChan:
				// Publish time up event
				s.Publisher.Publish(events.Event{
					Type:   events.EventTimeUp,
					GameID: s.ID.String(),
					Payload: messages.TimeupPayload{
						Color: string(color),
					},
				})
				s.Logger.Info("player time expired", zap.String("color", string(color)))
			}
		}
	}()
}

func (s *Game) Terminate() {
	close(s.done)
	s.Engine.Close()

	// Publish game terminated event
	s.Publisher.Publish(events.Event{
		Type:   events.EventGameTerminated,
		GameID: s.ID.String(),
		Payload: map[string]string{
			"game_id": s.ID.String(),
		},
	})
}
