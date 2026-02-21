package editor

type Mode int
type Key rune

const (
	ModeNormal Mode = iota
	ModeInsert
)

const TabSize = 4

// Control sequences
// Reference: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html (comments below to help search)
const (
	// CSI A
	KeyArrowUp = 1000 + iota
	// CSI B
	KeyArrowDown
	// CSI C
	KeyArrowRight
	// CSI D
	KeyArrowLeft
	// CSI 2 ~
	KeyInsert
	// CSI 3 ~
	KeyDelete
	// CSI 1 ~
	KeyHome
	// CSI 4 ~ OR CSI H
	KeyEnd
	// CSI 5 ~ OR CSI F
	KeyPageUp
	// CSI 6 ~
	KeyPageDown
)
