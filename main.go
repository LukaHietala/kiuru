package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gdamore/tcell/v3"
	"github.com/lukahietala/kiuru/buffer"
)

type Editor struct {
	events   chan tcell.Event
	screen   tcell.Screen
	quit     chan struct{}
	quitOnce sync.Once
}

func NewEditor() *Editor {
	return &Editor{
		events: make(chan tcell.Event, 4096),
		quit:   make(chan struct{}),
	}
}

func (e *Editor) Quit() {
	e.quitOnce.Do(func() {
		close(e.quit)
	})
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
	b := buffer.NewBuffer()

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

				// bloat, TODO: Make pasting actually faster
				for drain := true; drain; {
					select {
					case ev2 := <-e.events:
						isPasting = e.handleEvent(ev2, b, isPasting)
					default:
						drain = false
					}
				}
			}

			s.Clear()
			for i := 0; i < b.LineCount(); i++ {
				s.PutStr(0, i, string(b.LineBytes(i)))
			}
			cx, cy := b.Cursor()
			s.ShowCursor(cx, cy)
			s.Show()
		}
	}()

	<-ctx.Done()
	s.Fini()
	os.Exit(0)
}

func (e *Editor) handleEvent(ev tcell.Event, b *buffer.Buffer, isPasting bool) bool {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		e.screen.Sync()

	case *tcell.EventPaste:
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

		if ev.Key() == tcell.KeyBackspace {
			b.DeleteBack()
			return isPasting
		}

		if ev.Key() == tcell.KeyUp {
			b.MoveUp()
		}
		if ev.Key() == tcell.KeyDown {
			b.MoveDown()
		}
		if ev.Key() == tcell.KeyLeft {
			b.MoveLeft()
		}
		if ev.Key() == tcell.KeyRight {
			b.MoveRight()
		}
		// ...Switch cases are soy

		if ev.Key() == tcell.KeyRune {
			str := ev.Str()
			// TODO: dublication on WInDOWs rotta
			if str == "\n" || str == "\r" {
				b.InsertNewline()
			} else {
				b.InsertString(str)
			}
		}
	}

	return isPasting
}
