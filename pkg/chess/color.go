package chess

type Color string

const (
	White = "w"
	Black = "b"
)

func (c Color) Opp() Color {
	if c == White {
		return Black
	}

	return White
}
