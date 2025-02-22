package messages

import "encoding/json"

// InboundMessage is the generic wrapper for messages coming from the client.
// The "type" field tells us the action; "payload" is the data we parse further.
type InboundMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// StartNewGamePayload represents the payload for creating a new game
type StartNewGamePayload struct {
	TimeControl struct {
		WhiteTime      int64 `json:"white_time"`
		BlackTime      int64 `json:"black_time"`
		WhiteIncrement int64 `json:"white_increment"`
		BlackIncrement int64 `json:"black_increment"`
		MovesToGo      int64 `json:"moves_to_go"`
	} `json:"time_control"`
	Color string `json:"color"`
}

// MakeMovePayload represents the payload for making a move during a game
type MakeMovePayload struct {
	GameID string `json:"game_id"`
	Move   string `json:"move"`
}
