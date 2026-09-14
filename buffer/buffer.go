package buffer

import (
	"bytes"
	"path/filepath"

	"github.com/zeebo/xxh3"
)

type BufferFlag uint8

// Pretty much the same in vim
const (
	FlagReadonly BufferFlag = 1 << iota
	FlagScratch
	FlagNew
)

type Buffer struct {
	// TODO: Maybe rope later?
	name        string
	lines       [][]byte
	path        string
	initialHash uint64

	cursorX        int // byte offset!
	cursorY        int
	rowOff, colOff int

	flags BufferFlag

	undoTree UndoTree
}

func New() *Buffer {
	buf := &Buffer{
		lines: [][]byte{{}},
	}
	buf.initialHash = buf.CurrentHash()
	return buf
}

func NewScratch() *Buffer {
	buf := New()
	buf.MarkScratch()
	return buf
}

// Name returns the buffer name.
func (b *Buffer) Name() string {
	if b.HasFlag(FlagScratch) {
		return "Scratch"
	}
	if b.name == "" {
		return "No name"
	}
	return b.name
}

// Bytes returns a slice of bytes of all lines in a buffer separated by "\n".
// TODO: If dos then with \r\n
func (b *Buffer) Bytes() []byte {
	return bytes.Join(b.lines, []byte("\n"))
}

// LineBytes returns a slice of bytes of a specified line based on its index (position
// y in buffer).
func (b *Buffer) LineBytes(y int) []byte {
	if y < 0 || y >= len(b.lines) {
		return nil
	}
	return bytes.Clone(b.lines[y])
}

// LineCount returns the length of lines slice in a buffer.
func (b *Buffer) LineCount() int {
	return len(b.lines)
}

// CurrentHash returns buffer's xxh3 :delicious: hash
// TODO: If too large file do something else
func (b *Buffer) CurrentHash() uint64 {
	hasher := xxh3.New()

	for _, line := range b.lines {
		hasher.Write(line)
		hasher.Write([]byte{'\n'})
	}

	return hasher.Sum64()
}

// IsDirty returns true if buffer's initial hash doesn't match current hash
func (b *Buffer) IsDirty() bool {
	return b.CurrentHash() != b.initialHash
}

// SetPath updates the buffer's absolute path and name.
func (b *Buffer) SetPath(path string) {
	if abs, err := filepath.Abs(path); err == nil {
		b.path = abs
		b.name = filepath.Base(abs)
	} else {
		b.path = path
		b.name = filepath.Base(path)
	}
}

// MarkScratch changes buffer to scratch buffer
func (b *Buffer) MarkScratch() {
	b.path = ""
	b.name = ""
	b.EnableFlag(FlagScratch)
}

// Cursor returns the cursor's x and y pos (as rune index!) accounting for characters visual
// width
func (b *Buffer) Cursor() (int, int) {
	if b.cursorY < 0 || b.cursorY >= len(b.lines) {
		return 0, b.cursorY
	}
	return VisualWidth(b.lines[b.cursorY], b.cursorX), b.cursorY
}

// CursorBytes returns real byte offsets (x, y)
func (b *Buffer) CursorBytes() (int, int) {
	return b.cursorX, b.cursorY
}

// Offset returns row and col offsets (TODO): To window
func (b *Buffer) Offset() (int, int) {
	return b.rowOff, b.colOff
}

// SetOffset updates the row and col offsets
func (b *Buffer) SetOffset(rowOff, colOff int) {
	b.colOff = colOff
	b.rowOff = rowOff
}
