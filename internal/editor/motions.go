package editor

import "unicode"

type charClass int

const (
	classSpace charClass = iota
	classWord
	classPunct
)

// Group runes, very simple for now
// Vim: :help character-classes
func getCharClass(r rune) charClass {
	if unicode.IsSpace(r) {
		return classSpace
	}
	// Vim "iskeyword": @,48-57,192-255,$,_
	// This is not exactly same as it includes all characters in category L
	// https://util.unicode.org/UnicodeJsps/list-unicodeset.jsp?a=[%3AL%3A], query: [:L:]
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' {
		return classWord
	}
	// TODO?: Edge cases?
	return classPunct
}

// Move cursor left by one
func (b *Buffer) MoveLeft() {
	if b.Cx > 0 {
		b.Cx--
		b.DesiredCx = b.Cx
	}
}

// Move cursor down by one
func (b *Buffer) MoveDown() {
	if b.Cy < len(b.Rows)-1 {
		b.Cy++
		b.ClampCursor()
	}
}

// Move cursor up by one
func (b *Buffer) MoveUp() {
	if b.Cy > 0 {
		b.Cy--
		b.ClampCursor()
	}
}

// Move cursor right by one
func (b *Buffer) MoveRight() {
	if b.Cy < len(b.Rows) && b.Cx < len(b.Rows[b.Cy]) {
		b.Cx++
		b.DesiredCx = b.Cx
	}
}

// "PageUp", moves up by screen height
func (b *Buffer) PageUp(termRows int) {
	b.Cy = max(0, b.Cy-termRows)
	b.ClampCursor()
}

// "PageDown", moves down by screen height
func (b *Buffer) PageDown(termRows int) {
	b.Cy = min(len(b.Rows)-1, b.Cy+termRows)
	b.ClampCursor()
}

// "<Home>" Move to line to beginning
// TODO: Stop at whitespace
func (b *Buffer) MoveStart() {
	b.Cx = 0
	b.DesiredCx = b.Cx
}

// "$", "<End>" Move to line end
// TODO: Stop at whitespace
func (b *Buffer) MoveEnd() {
	b.Cx = len(b.Rows[b.Cy])
	b.DesiredCx = b.Cx
}

// "^" Move to next non-space
func (b *Buffer) MoveFirstNonBlank() {
	if len(b.Rows) == 0 {
		return
	}
	if b.Cy >= len(b.Rows) {
		b.Cy = len(b.Rows) - 1
	}
	row := b.Rows[b.Cy]
	b.Cx = 0
	for b.Cx < len(row) && getCharClass(row[b.Cx]) == classSpace {
		b.Cx++
	}
	b.DesiredCx = b.Cx
}

// "w" Move forward one word
func (b *Buffer) MoveWord() {
	if len(b.Rows) == 0 {
		return
	}
	b.Cx++

	for b.Cy < len(b.Rows) {
		if b.Cx >= len(b.Rows[b.Cy]) {
			b.Cy++
			b.Cx = 0
			if b.Cy < len(b.Rows) && len(b.Rows[b.Cy]) == 0 {
				break
			}
			continue
		}

		// Vim likes to land only on non-space chars
		if getCharClass(b.Rows[b.Cy][b.Cx]) != classSpace {
			prevCx, prevCy := b.Cx-1, b.Cy
			if prevCx < 0 {
				break
			}
			if getCharClass(b.Rows[prevCy][prevCx]) != getCharClass(b.Rows[b.Cy][b.Cx]) {
				break
			}
		}
		b.Cx++
	}

	// Bounds check
	if b.Cy >= len(b.Rows) {
		b.Cy = len(b.Rows) - 1
		b.Cx = len(b.Rows[b.Cy])
	}
	b.DesiredCx = b.Cx
}

// "b" Move backward one word
func (b *Buffer) MoveBack() {
	if b.Cx == 0 && b.Cy == 0 {
		return
	}
	b.Cx--

	for {
		if b.Cx < 0 {
			if b.Cy == 0 {
				b.Cx = 0
				break
			}
			b.Cy--
			b.Cx = len(b.Rows[b.Cy]) - 1
			if len(b.Rows[b.Cy]) == 0 {
				b.Cx = 0
				break
			}
		}

		if getCharClass(b.Rows[b.Cy][b.Cx]) != classSpace {
			targetClass := getCharClass(b.Rows[b.Cy][b.Cx])
			for b.Cx > 0 && getCharClass(b.Rows[b.Cy][b.Cx-1]) == targetClass {
				b.Cx--
			}
			break
		}
		b.Cx--
	}
	b.DesiredCx = b.Cx
}

// "e" Move to end of next word
func (b *Buffer) MoveEndWord() {
	if len(b.Rows) == 0 {
		return
	}
	b.Cx++

	for b.Cy < len(b.Rows) {
		if b.Cx >= len(b.Rows[b.Cy]) {
			b.Cy++
			b.Cx = 0
			if b.Cy >= len(b.Rows) {
				break
			}
			if len(b.Rows[b.Cy]) == 0 {
				break
			}
		}

		if getCharClass(b.Rows[b.Cy][b.Cx]) == classSpace {
			b.Cx++
			continue
		}

		targetClass := getCharClass(b.Rows[b.Cy][b.Cx])
		for b.Cx+1 < len(b.Rows[b.Cy]) && getCharClass(b.Rows[b.Cy][b.Cx+1]) == targetClass {
			b.Cx++
		}
		break
	}

	// Bounds check
	if b.Cy >= len(b.Rows) {
		b.Cy = len(b.Rows) - 1
		b.Cx = len(b.Rows[b.Cy]) - 1
	}
	b.DesiredCx = b.Cx
}
