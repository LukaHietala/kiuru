package main

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell/v3"
)

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
				s.Put(0, 0, ev.Str(), tcell.StyleDefault)
			}
		case *tcell.EventMouse:
			x, y := ev.Position()
			s.PutStr(0, 0, fmt.Sprintf("Rotta: %d %d", x, y))
		}
	}
}
