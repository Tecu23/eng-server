// Package color provides basic color definitions for a chess game
package color

// Color represent a chess color
type Color string

// Possible color variations in a chess game
const (
	White = "w"
	Black = "b"
)

// Opp returns the opposite color for the given color.
func (c Color) Opp() Color {
	if c == White {
		return Black
	}

	return White
}
