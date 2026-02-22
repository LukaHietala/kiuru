package editor

import (
	"sync"
	"unicode"

	"github.com/mattn/go-runewidth"
)

type Editor struct {
	mu       sync.Mutex
	Mode     Mode
	Quit     bool
	Buffers  []*Buffer
	BufIndex int
	Debug    bool

	// Usually stdin
	input  InputReader
	render Renderer
}

func NewEditor(in InputReader, render Renderer, debug bool) *Editor {
	e := &Editor{
		Mode:    ModeNormal,
		Buffers: []*Buffer{},
		Debug:   debug,
		input:   in,
		render:  render,
	}
	e.render.UpdateSize()
	return e
}

func (e *Editor) AddBuffer(b *Buffer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Buffers = append(e.Buffers, b)
	e.BufIndex = len(e.Buffers) - 1
}

func (e *Editor) CurBuf() *Buffer {
	if len(e.Buffers) == 0 {
		return nil
	}
	return e.Buffers[e.BufIndex]
}

// TODO: MOVE TO SOMEWHERE ELSE, make public versions with locking
func (e *Editor) nextBuf() {
	if e.BufIndex >= len(e.Buffers)-1 {
		return
	}
	e.BufIndex++
	if b := e.CurBuf(); b != nil {
		b.ClampCursor()
	}
}

func (e *Editor) prevBuf() {
	if e.BufIndex <= 0 {
		return
	}
	e.BufIndex--
	if b := e.CurBuf(); b != nil {
		b.ClampCursor()
	}
}

func (e *Editor) UpdateSize() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.render.UpdateSize()
	if b := e.CurBuf(); b != nil {
		if b.Cy >= len(b.Rows) {
			b.Cy = max(0, len(b.Rows)-1)
		}
		b.ClampCursor()
	}
}

func (e *Editor) Draw() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.scroll()
	e.render.Render(e.CurBuf(), e.Mode, e.Debug)
}

func (e *Editor) ProcessKey(key Key) {
	e.mu.Lock()
	defer e.mu.Unlock()

	b := e.CurBuf()
	if b == nil {
		return
	}

	switch e.Mode {
	case ModeNormal:
		switch key {
		case ']':
			e.nextBuf()
		case '[':
			e.prevBuf()
		case 'q':
			e.Quit = true
		case 'i':
			e.Mode = ModeInsert
		case 'v':
			e.Mode = ModeVisual
			b.Selection.Start = Mark{X: b.Cx, Y: b.Cy}
			b.Selection.Active = true
		case 'h', KeyArrowLeft:
			e.moveLeft(b)
		case 'j', KeyArrowDown:
			e.moveDown(b)
		case 'k', KeyArrowUp:
			e.moveUp(b)
		case 'l', KeyArrowRight:
			e.moveRight(b)
		case 'G':
			b.Cy = len(b.Rows) - 1
			b.ClampCursor()
		case KeyPageUp:
			e.pageUp(b)
		case KeyPageDown:
			e.pageDown(b)
		case KeyHome:
			e.moveStart(b)
		case KeyEnd:
			e.moveEnd(b)
		case KeyDelete, 'x':
			b.DeleteChar(false)
		}
	case ModeInsert:
		switch key {
		case 27: // Escape
			e.Mode = ModeNormal
		case 13: // Enter
			b.InsertNewline()
		case 127: // Backspace
			b.DeleteChar(true)
		case KeyDelete:
			b.DeleteChar(false)
		case '\t':
			b.InsertChar('\t')
		case KeyArrowLeft:
			e.moveLeft(b)
		case KeyArrowDown:
			e.moveDown(b)
		case KeyArrowUp:
			e.moveUp(b)
		case KeyArrowRight:
			e.moveRight(b)
		case KeyPageUp:
			e.pageUp(b)
		case KeyPageDown:
			e.pageDown(b)
		case KeyHome:
			e.moveStart(b)
		case KeyEnd:
			e.moveEnd(b)
		default:
			if unicode.IsPrint(rune(key)) {
				b.InsertChar(rune(key))
			}
		}
	case ModeVisual:
		switch key {
		case 27, 'v':
			e.Mode = ModeNormal
			b.Selection.Clear()
		case 'd', 'x', KeyDelete:
			b.DeleteSelection()
			e.Mode = ModeNormal
		case 'h', KeyArrowLeft:
			e.moveLeft(b)
		case 'j', KeyArrowDown:
			e.moveDown(b)
		case 'k', KeyArrowUp:
			e.moveUp(b)
		case 'l', KeyArrowRight:
			e.moveRight(b)
		case KeyPageUp:
			e.pageUp(b)
		case KeyPageDown:
			e.pageDown(b)
		case KeyHome:
			e.moveStart(b)
		case KeyEnd:
			e.moveEnd(b)
		}
	}
}

func (e *Editor) moveLeft(b *Buffer) {
	if b.Cx > 0 {
		b.Cx--
		b.DesiredCx = b.Cx
	}
}

func (e *Editor) moveDown(b *Buffer) {
	if b.Cy < len(b.Rows)-1 {
		b.Cy++
		b.ClampCursor()
	}
}

func (e *Editor) moveUp(b *Buffer) {
	if b.Cy > 0 {
		b.Cy--
		b.ClampCursor()
	}
}

func (e *Editor) moveRight(b *Buffer) {
	if b.Cy < len(b.Rows) && b.Cx < len(b.Rows[b.Cy]) {
		b.Cx++
		b.DesiredCx = b.Cx
	}
}

func (e *Editor) pageUp(b *Buffer) {
	_, termRows := e.render.Size()
	b.Cy = max(0, b.Cy-termRows)
	b.ClampCursor()
}

func (e *Editor) pageDown(b *Buffer) {
	_, termRows := e.render.Size()
	b.Cy = min(len(b.Rows)-1, b.Cy+termRows)
	b.ClampCursor()
}

func (e *Editor) moveStart(b *Buffer) {
	b.Cx = 0
	b.DesiredCx = b.Cx
}

func (e *Editor) moveEnd(b *Buffer) {
	b.Cx = len(b.Rows[b.Cy])
	b.DesiredCx = b.Cx
}

func (e *Editor) scroll() {
	// TODO: (bug) cursor offsets on horizontal scroll if previous is 2-width char
	b := e.CurBuf()
	cols, rows := e.render.Size()

	if b.Cy < b.RowOff {
		b.RowOff = b.Cy
	}
	if b.Cy >= b.RowOff+rows {
		b.RowOff = b.Cy - rows + 1
	}

	b.Vx = 0
	if b.Cy < len(b.Rows) {
		row := b.Rows[b.Cy]
		for i := 0; i < b.Cx && i < len(row); i++ {
			char := row[i]
			if char == '\t' {
				b.Vx += (TabSize - 1) - (b.Vx % TabSize)
				b.Vx++
			} else if char == '\r' {
				b.Vx += 2
			} else {
				b.Vx += runewidth.RuneWidth(char)
			}
		}
	}

	gutterWidth := getGutterWidth(b)
	termWidth := cols - gutterWidth

	if b.Vx < b.ColOff {
		b.ColOff = b.Vx
	}
	if b.Vx >= b.ColOff+termWidth {
		b.ColOff = b.Vx - termWidth + 1
	}
}

func (e *Editor) ShouldQuit() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.Quit
}
