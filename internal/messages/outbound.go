package messages

import "github.com/tecu23/eng-server/pkg/chess"

// OutboundMessage is how we wrap responses before sending
// them to the client
type OutboundMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
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
	CurrentTurn chess.Color `json:"current_turn"`
}

type GameOverPayload struct {
	Reason string `json:"reason"`
}

// GameStatePayload represents the payload returned after updating the game state
type GameStatePayload struct {
	GameID      string      `json:"game_id"`
	BoardFEN    string      `json:"board_fen"`
	WhiteTime   int64       `json:"white_time"`
	BlackTime   int64       `json:"black_time"`
	CurrentTurn chess.Color `json:"current_turn"`
	IsCheckmate bool        `json:"is_checkmate"`
	IsDraw      bool        `json:"is_draw"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type EngineMovePayload struct {
	Move  string      `json:"move"`
	Color chess.Color `json:"color"`
}

// ClockUpdatePayload contains information about the current state of the clock
type ClockUpdatePayload struct {
	WhiteTime   int64  `json:"whiteTimeMs"` // White's remaining time in milliseconds
	BlackTime   int64  `json:"blackTimeMs"` // Black's remaining time in milliseconds
	ActiveColor string `json:"activeColor"` // The color of the active player (White or Black)
}

// TimeupPayload contains information about which player ran out of time
type TimeupPayload struct {
	Color string `json:"color"` // The color of the player who ran out of time
}
