package messages

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
	GameID      string `json:"game_id"`
	InitialFEN  string `json:"initial_fen"`
	WhiteTime   int64  `json:"white_time"`
	BlackTime   int64  `json:"black_time"`
	CurrentTurn string `json:"current_turn"`
}

type GameOverPayload struct {
	Reason string `json:"reason"`
}

type TimeUpdatePayload struct {
	Color     string `json:"color"`
	Remaining int64  `json:"remaining"`
}

// GameStatePayload represents the payload returned after updating the game state
type GameStatePayload struct {
	GameID      string `json:"game_id"`
	BoardFEN    string `json:"board_fen"`
	WhiteTime   int64  `json:"white_time"`
	BlackTime   int64  `json:"black_time"`
	CurrentTurn string `json:"current_turn"`
	IsCheckmate bool   `json:"is_checkmate"`
	IsDraw      bool   `json:"is_draw"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type EngineMovePayload struct {
	Move  string `json:"move"`
	Color string `json:"color"`
}
