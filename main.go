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
				// Tcell desided to remove syncronous polling so
				// go channel overhead is forced. Maybe raw TTY later.
				// TODO: Maybe something like: TTY -> Pasting -> To EventQ ->
				// Other events?
				for drain := true; drain; {
					select {
					case ev2 := <-e.events:
						isPasting = e.handleEvent(ev2, b, isPasting)
					default:
						drain = false
					}
				}
			}

			if isPasting {
				continue
			}

			w, h := s.Size()
			cx, cy := b.Cursor()
			rowOff, colOff := b.Offset()

			// Clamp offsets so cursor stays on the screen
			rowOff = max(cy-h+1, min(rowOff, cy))
			colOff = max(cx-w+1, min(colOff, cx))
			b.SetOffset(rowOff, colOff)

			// Render only visible lines
			s.Clear()
			for y := range h {
				line := y + rowOff
				if line >= b.LineCount() {
					break
				}

				runes := []rune(expandTabs(b.LineBytes(line)))
				if colOff < len(runes) {
					s.PutStr(0, y, string(runes[colOff:]))
				}
			}

			s.ShowCursor(cx-colOff, cy-rowOff)
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
		w, h := ev.Size()
		log.Printf("%d, %d", w, h)
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
		}

		if ev.Key() == tcell.KeyEnter || ev.Key() == tcell.KeyCtrlJ {
			b.InsertNewline()
		}

		if ev.Key() == tcell.KeyBackspace {
			b.DeleteBack()
		}

		if ev.Key() == tcell.KeyDelete {
			b.DeleteForward()
		}

		if ev.Key() == tcell.KeyTab {
			b.InsertString("\t")
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

// TODO: to rendering
func expandTabs(line []byte) string {
	var out []rune
	col := 0
	for _, r := range string(line) {
		if r == '\t' {
			spaces := 4 - (col % 4)
			for range spaces {
				out = append(out, ' ')
				col++
			}
		} else {
			out = append(out, r)
			col++
		}
	}
	return string(out)
}
