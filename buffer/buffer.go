package buffer

import "bytes"

type Buffer struct {
	lines   [][]byte
	cursorX int // byte offset
	cursorY int
}

func NewBuffer() *Buffer {
	return &Buffer{lines: [][]byte{{}}}
}

func (b *Buffer) Bytes() []byte {
	// TODO: CRLF
	return bytes.Join(b.lines, []byte("\n"))
}

func (b *Buffer) LineBytes(y int) []byte {
	// TODO: safeguard other helpers for future api?
	if y < 0 || y >= len(b.lines) {
		return nil
	}
	return b.lines[y]
}

func (b *Buffer) LineCount() int {
	return len(b.lines)
}

// For visual, accounts for different width chars like CJK
func (b *Buffer) Cursor() (int, int) {
	return VisualWidth(b.lines[b.cursorY], b.cursorX), b.cursorY
}
