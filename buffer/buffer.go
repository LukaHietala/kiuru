package buffer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type BufferFlag uint8

// Pretty much the same in vim
const (
	FlagReadonly BufferFlag = 1 << iota
	FlagScratch
	FlagModifiable
)

type Buffer struct {
	// TODO: Maybe rope later?
	name  string
	lines [][]byte
	path  string

	cursorX        int // byte offset!
	cursorY        int
	rowOff, colOff int

	flags BufferFlag
}

// TODO: modifiable
func NewBuffer(path string, scratch bool, readonly bool) (*Buffer, error) {
	buf := &Buffer{
		lines: [][]byte{{}},
		flags: FlagModifiable,
	}

	if scratch {
		buf.MarkScratch()
		return buf, nil
	}

	if path == "" {
		if readonly {
			buf.EnableFlag(FlagReadonly)
		}
		return buf, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	buf.path = absPath
	buf.name = filepath.Base(absPath)

	if readonly {
		buf.EnableFlag(FlagReadonly)
	}

	absPath, isReadOnly, err := validatePath(path)
	if err != nil {
		if os.IsNotExist(err) {
			return buf, nil
		}
		buf.DisableFlag(FlagModifiable)
		return buf, err
	}

	buf.path = absPath
	buf.name = filepath.Base(absPath)
	if isReadOnly {
		buf.EnableFlag(FlagReadonly)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// TODO: keep track if dos
	content = bytes.ReplaceAll(content, []byte("\r"), []byte(""))
	buf.lines = bytes.Split(content, []byte("\n"))

	return buf, nil
}

// Returns buffer name
func (b *Buffer) Name() string {
	if b.HasFlag(FlagScratch) {
		return "Scratch"
	}
	if b.name == "" {
		return "No name"
	}
	return b.name
}

// Bytes returns a slice of bytes of all lines in a buffer sperated by "\n".
// TODO: If dos then with /r/n
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
	if b.cursorY < 0 || b.cursorY >= len(b.lines) {
		return 0, b.cursorY
	}
	return VisualWidth(b.lines[b.cursorY], b.cursorX), b.cursorY
}

// MarkScratch changes buffer to scratch buffer
func (b *Buffer) MarkScratch() {
	b.path = ""
	b.name = ""
	b.EnableFlag(FlagScratch)
	b.EnableFlag(FlagModifiable)
}

// CursorBytes returns real byte offsets (x, y)
func (b *Buffer) CursorBytes() (int, int) {
	return b.cursorX, b.cursorY
}

// HasFlag checks if flag is enabled
func (b *Buffer) HasFlag(f BufferFlag) bool {
	return b.flags&f != 0
}

// EnableFlag enables a specific flag
func (b *Buffer) EnableFlag(f BufferFlag) {
	b.flags |= f
}

// DisableFlag disables a specific flag
func (b *Buffer) DisableFlag(f BufferFlag) {
	b.flags &^= f
}

// ToggleFlag flips a specific flag between enabled and disabled
func (b *Buffer) ToggleFlag(f BufferFlag) {
	b.flags ^= f
}

// Validates path and its contents
func validatePath(path string) (string, bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", false, err
	}

	if info.IsDir() {
		// TODO: Open file explorer
		return "", false, fmt.Errorf("'%s' is a directory", path)
	}

	// TODO: Ignores group perms and might not work for every windows case
	var readonly bool
	if info.Mode().Perm()&0o200 == 0 {
		// TODO: Warn
		readonly = true
	}

	return abs, readonly, nil
}
