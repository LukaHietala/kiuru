package buffer

import (
	"slices"
)

// TODO: Guard everything in the most paranoid way possible

func (b *Buffer) InsertString(s string) {
	if len(s) == 0 {
		return
	}
	b.lines[b.cursorY] = slices.Insert(b.lines[b.cursorY], b.cursorX, []byte(s)...)
	b.cursorX += len(s)
}

func (b *Buffer) InsertNewline() {
	line := b.lines[b.cursorY]
	remainder := slices.Clone(line[b.cursorX:])
	b.lines[b.cursorY] = slices.Clone(line[:b.cursorX])
	b.lines = slices.Insert(b.lines, b.cursorY+1, remainder)
	b.cursorY++
	b.cursorX = 0
}

func (b *Buffer) DeleteBack() {
	line := b.lines[b.cursorY]
	if b.cursorX > 0 {
		size := PrevRuneSize(line, b.cursorX)
		b.lines[b.cursorY] = slices.Delete(line, b.cursorX-size, b.cursorX)
		b.cursorX -= size
	} else if b.cursorY > 0 {
		prevLen := len(b.lines[b.cursorY-1])
		b.lines[b.cursorY-1] = append(b.lines[b.cursorY-1], b.lines[b.cursorY]...)
		b.lines = slices.Delete(b.lines, b.cursorY, b.cursorY+1)
		b.cursorY--
		b.cursorX = prevLen
	}
}

func (b *Buffer) DeleteForward() {
	line := b.lines[b.cursorY]
	if b.cursorX < len(line) {
		size := NextRuneSize(line, b.cursorX)
		b.lines[b.cursorY] = slices.Delete(line, b.cursorX, b.cursorX+size)
	} else if b.cursorY+1 < len(b.lines) {
		b.lines[b.cursorY] = append(b.lines[b.cursorY], b.lines[b.cursorY+1]...)
		b.lines = slices.Delete(b.lines, b.cursorY+1, b.cursorY+2)
	}
}
