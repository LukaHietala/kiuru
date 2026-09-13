package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
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
	// TEMP
	conn, err := net.Dial("tcp", "127.0.0.1:5000")
	if err == nil {
		log.SetOutput(conn)
	}

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
	ml := NewMessageLog()
	flag.Parse()
	args := flag.Args()

	var path string

	if len(args) > 0 {
		path = args[0]
	}

	b, err := buffer.OpenFile(path)
	if err != nil {
		ml.Post(err.Error())
		b = buffer.New()
		b.SetPath(path)
		b.EnableFlag(buffer.FlagReadonly)
	}
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

			gutterW := len(fmt.Sprintf("%d", b.LineCount()))
			contentW := w - gutterW - 1

			// Clamp offsets so cursor stays on the screen, accounting for gutter width
			rowOff = max(cy-h+3, min(rowOff, cy))
			if contentW > 0 {
				colOff = max(cx-contentW+1, min(colOff, cx))
			} else {
				colOff = 0
			}
			b.SetOffset(rowOff, colOff)

			// Render only visible lines
			s.Clear()
			for y := range h - 2 {
				line := y + rowOff
				if line >= b.LineCount() {
					s.PutStrStyled(0, y, "~", tcell.StyleDefault.Foreground(color.DimGray))
					continue
				}

				numStyle := tcell.StyleDefault.Foreground(color.DimGray)
				if cy == line {
					numStyle = numStyle.Foreground(color.LightPink)
				}
				s.PutStrStyled(0, y, fmt.Sprintf("%*d ", gutterW, line+1), numStyle)

				runes := []rune(expandTabs(b.LineBytes(line)))
				if colOff < len(runes) {
					s.PutStr(gutterW+1, y, string(runes[colOff:]))
				}
			}

			// TODO: Very dirty, move away
			cxb, _ := b.CursorBytes()
			statusStyle := tcell.StyleDefault.Foreground(color.Black).Background(color.White)

			left := " " + b.Name()
			if flags := b.Flags().String(); flags != "" {
				left = fmt.Sprintf(" %s (%s)", b.Name(), flags)
			}

			// Only col should have differing byte and rune offsets
			colStr := fmt.Sprintf("%d", cx+1)
			if cxb != cx {
				colStr = fmt.Sprintf("%d-%d", cxb+1, cx+1)
			}

			lineStr := fmt.Sprintf("%d", cy+1)
			right := fmt.Sprintf("L: %s C: %s ", lineStr, colStr)

			middleW := w - len(left) - len(right)
			fullStr := fmt.Sprintf("%s%-*s%s", left, middleW, "", right)
			s.PutStrStyled(0, h-2, fullStr, statusStyle)
			s.PutStrStyled(0, h-1, fmt.Sprintf("%-*s", w, ""), tcell.StyleDefault.Background(color.Black))
			s.PutStr(0, h-1, fmt.Sprintf("%-*s", w, ml.Current()))

			s.ShowCursor(gutterW+1+cx-colOff, cy-rowOff)
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
		if ev.Key() == tcell.KeyCtrlQ {
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
