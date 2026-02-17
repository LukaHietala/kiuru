package ansi

import "strconv"

const (
	HideCursor   = "\x1b[?25l"
	ShowCursor   = "\x1b[?25h"
	ClearLine    = "\x1b[K"
	CursorHome   = "\x1b[H"
	AltBufferOn  = "\x1b[?1049h"
	AltBufferOff = "\x1b[?1049l"
	ReverseVideo = "\x1b[7m"
	ResetFormat  = "\x1b[m"
	ClearScreen  = "\x1b[2J"
)

func MoveCursor(y, x int) string {
	return "\x1b[" + strconv.Itoa(y) + ";" + strconv.Itoa(x) + "H"
}
