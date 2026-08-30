package main

import "slices"

type Buffer struct {
	Lines            [][]rune
	CursorX, CursorY int
}

func NewBuffer() *Buffer {
	return &Buffer{
		Lines: [][]rune{{}},
	}
}

func (b *Buffer) InsertChar(runes []rune) {
	if len(runes) == 0 {
		return
	}
	b.Lines[b.CursorY] = slices.Insert(b.Lines[b.CursorY], b.CursorX, runes...)
	b.CursorX += len(runes)
}

func (b *Buffer) InsertNewline() {
	line := b.Lines[b.CursorY]
	remainder := slices.Clone(line[b.CursorX:])
	b.Lines[b.CursorY] = line[:b.CursorX]
	b.Lines = slices.Insert(b.Lines, b.CursorY+1, remainder)
	b.CursorY++
	// TODO: Move next to tab
	b.CursorX = 0
}
