package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell/v3"
)

type Editor struct {
	events chan tcell.Event
	screen tcell.Screen
	quit   chan struct{}
}

func NewEditor() *Editor {
	return &Editor{
		events: make(chan tcell.Event, 4096),
		quit:   make(chan struct{}),
	}
}

func (e *Editor) Quit() {
	// TODO: max once
	close(e.quit)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}

	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	defer func() {
		r := recover()
		s.Fini()
		if r != nil {
			panic(r)
		}
	}()

	e := NewEditor()
	b := NewBuffer()

	e.screen = s

	// Poll tcell events
	go func() {
		for ev := range s.EventQ() {
			e.events <- ev
		}
	}()

	// Render loop (for now)
	go func() {
		isPasting := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-e.quit:
				stop()
				return
			case ev := <-e.events:
				isPasting = e.handleEvent(ev, b, isPasting)

				// If multiple events came in. Paste for example
				// Go through them before rendering
				for drain := true; drain; {
					select {
					case ev2 := <-e.events:
						isPasting = e.handleEvent(ev2, b, isPasting)
					default:
						drain = false
					}
				}
			}

			for i, line := range b.Lines {
				s.PutStr(0, i, string(line))
			}
			s.ShowCursor(b.CursorX, b.CursorY)
			s.Show()
		}
	}()

	<-ctx.Done()
	s.Fini()
	os.Exit(0)
}

func (e *Editor) handleEvent(ev tcell.Event, b *Buffer, isPasting bool) bool {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		e.screen.Sync()

	case *tcell.EventPaste:
		// Bracketed paste
		if ev.Start() {
			isPasting = true
		} else if ev.End() {
			isPasting = false
		}

	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
			e.Quit()
			return isPasting
		}

		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyCtrlJ {
			b.InsertNewline()
			return isPasting
		}

		if ev.Key() == tcell.KeyRune {
			str := ev.Str()
			// TODO: Now it inserts twice BECAUSE OF WINDOWS \r\n
			// Too lazy to handle it now
			if str == "\n" || str == "\r" {
				b.InsertNewline()
			} else {
				b.InsertChar([]rune(str))
			}
		}
	}

	return isPasting
}
