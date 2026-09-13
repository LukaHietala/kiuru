package buffer

import "bytes"

type Buffer struct {
	// TODO: Maybe rope later?
	lines          [][]byte
	cursorX        int // byte offset!
	cursorY        int
	rowOff, colOff int
}

func NewBuffer() *Buffer {
	return &Buffer{lines: [][]byte{{}}}
}

// Bytes returns a slice of bytes of all lines in a buffer sperated by "\n".
func (b *Buffer) Bytes() []byte {
	return bytes.Join(b.lines, []byte("\n"))
}

// LineBytes returns a slice of bytes of a specified line based on its index (position
// y in buffer).
func (b *Buffer) LineBytes(y int) []byte {
	if y < 0 || y >= len(b.lines) {
		return nil
	}
	return b.lines[y]
}

// LineCount returns the length of lines slice in a buffer.
func (b *Buffer) LineCount() int {
	return len(b.lines)
}

// Offset returns row and col offsets (TODO): To window
func (b *Buffer) Offset() (int, int) {
	return b.rowOff, b.colOff
}

func (b *Buffer) SetOffset(rowOff, colOff int) {
	b.colOff = colOff
	b.rowOff = rowOff
}

// Cursor returns the cursor's x and y pos (as rune index!) accounting for characters visual
// width
func (b *Buffer) Cursor() (int, int) {
	return VisualWidth(b.lines[b.cursorY], b.cursorX), b.cursorY
}

// CursorBytes returns real byte offsets (x, y)
func (b *Buffer) CursorBytes() (int, int) {
	return b.cursorX, b.cursorY

}
