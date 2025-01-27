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
		Initial   int `json:"initial"`
		Increment int `json:"increment"`
	} `json:"time_control"`
	Color string `json:"color"`
}

// MakeMovePayload represents the payload for making a move during a game
type MakeMovePayload struct {
	GameID string `json:"game_id"`
	Move   string `json:"move"`
}
