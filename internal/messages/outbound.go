package messages

import (
	"github.com/tecu23/eng-server/internal/color"
)

// OutboundMessage is how we wrap responses before sending
// them to the client
type OutboundMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// ClockUpdatePayload contains information about the current state of the clock
type ClockUpdatePayload struct {
	GameID      string `json:"gameId"`
	WhiteTime   int64  `json:"whiteTimeMs"`
	BlackTime   int64  `json:"blackTimeMs"`
	ActiveColor string `json:"activeColor"`
}

// GameOverPayload contains the information about the state on an ended game
type GameOverPayload struct {
	GameID      string `json:"gameId"`
	Reason      string `json:"reason"`
	Result      string `json:"result"`
	Description string `json:"description"`
}

// Resignation payload
type ResignPayload struct {
	GameID string `json:"gameId"`
}

type ConnectedPayload struct {
	ConnectionId string `json:"connection_id"`
}

// GameCreatedPayload represents the payload after a create game event
type GameCreatedPayload struct {
	GameID      string      `json:"game_id"`
	InitialFEN  string      `json:"initial_fen"`
	WhiteTime   int64       `json:"white_time"`
	BlackTime   int64       `json:"black_time"`
	CurrentTurn color.Color `json:"current_turn"`
}

// GameStatePayload represents the payload returned after updating the game state
type GameStatePayload struct {
	GameID      string      `json:"game_id"`
	BoardFEN    string      `json:"board_fen"`
	WhiteTime   int64       `json:"white_time"`
	BlackTime   int64       `json:"black_time"`
	CurrentTurn color.Color `json:"current_turn"`
	IsCheckmate bool        `json:"is_checkmate"`
	IsDraw      bool        `json:"is_draw"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type EngineMovePayload struct {
	Move  string      `json:"move"`
	Color color.Color `json:"color"`
}

// TimeupPayload contains information about which player ran out of time
type TimeupPayload struct {
	Color string `json:"color"` // The color of the player who ran out of time
}
