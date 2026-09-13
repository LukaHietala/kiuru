package buffer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	name  string
	lines [][]byte
	path  string

	cursorX        int // byte offset!
	cursorY        int
	rowOff, colOff int

	flags BufferFlag
}

func New() *Buffer {
	return &Buffer{
		lines: [][]byte{{}},
	}
}

func NewScratch() *Buffer {
	buf := New()
	buf.MarkScratch()
	return buf
}

func OpenFile(path string) (*Buffer, error) {
	if path == "" {
		return New(), nil
	}

	buf := New()

	absPath, isReadOnly, err := validatePath(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if abs, absErr := filepath.Abs(path); absErr == nil {
				path = abs
			}
			buf.path = path
			buf.name = filepath.Base(path)
			buf.EnableFlag(FlagNew)
			return buf, nil
		}
		return nil, err
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
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	buf.lines = bytes.Split(content, []byte("\n"))

	return buf, nil
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

func (b *Buffer) Flags() BufferFlag {
	return b.flags
}

func (b *Buffer) HasFlag(f BufferFlag) bool {
	return b.flags&f != 0
}

func (b *Buffer) EnableFlag(f BufferFlag) {
	b.flags |= f
}

func (b *Buffer) DisableFlag(f BufferFlag) {
	b.flags &^= f
}

func (b *Buffer) ToggleFlag(f BufferFlag) {
	b.flags ^= f
}

func (f BufferFlag) String() string {
	if f == 0 {
		return ""
	}

	var parts []string
	if f&FlagNew != 0 {
		parts = append(parts, "new")
	}
	if f&FlagReadonly != 0 {
		parts = append(parts, "readonly")
	}
	// No scratch because it's already in the name :D

	return strings.Join(parts, "|")
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

	// TODO: too limiting?
	// https://pkg.go.dev/io/fs#FileMode
	if !info.Mode().IsRegular() {
		return "", true, fmt.Errorf("'%s' is not a regular file", path)
	}

	// TODO: Ignores group perms and might not work for every windows case
	var readonly bool
	if info.Mode().Perm()&0o200 == 0 {
		// TODO: Warn
		readonly = true
	}

	return abs, readonly, nil
}
