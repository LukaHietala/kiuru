package main

import (
	"fmt"
	"log"
	"slices"

	"github.com/gdamore/tcell/v3"
)

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

func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}
	s.EnableMouse()
	s.Clear()

	defer func() {
		r := recover()
		s.Fini()
		if r != nil {
			panic(r)
		}
	}()

	b := NewBuffer()

	for {
		s.Show()

		ev := <-s.EventQ()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return
			}
			if ev.Key() == tcell.KeyRune {
				b.InsertChar([]rune(ev.Str()))
				for _, line := range b.Lines {
					s.PutStr(0, 0, string(line))
				}
			}
		case *tcell.EventMouse:
			x, y := ev.Position()
			s.PutStr(0, 0, fmt.Sprintf("Rotta: %d %d", x, y))
		}
	}
}
