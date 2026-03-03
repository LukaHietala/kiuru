package editor

import (
	"slices"
	"sync"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// TODO: Complete mess

type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
)

const TabSize = 4

type Editor struct {
	mu       sync.Mutex
	Mode     Mode
	Quit     bool
	Buffers  []*Buffer
	BufIndex int
	Debug    bool

	Count         int // Multiplier (3 in 3j)
	PendingOp     Key // Pending operation (d in d3j)
	PendingMotion Key // Pending motion (g in gg)

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

// TODO: Make public ones with locking
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

func (e *Editor) scroll() {
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
			// TODO: add other garbage from renderer
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

func ctrlKey(key byte) Key {
	return Key(key & 0x1f)
}

func (e *Editor) multiplier() int {
	if e.Count == 0 {
		return 1
	}
	return e.Count
}

func (e *Editor) clearState() {
	e.Count = 0
	e.PendingOp = 0
	e.PendingMotion = 0
}

func (e *Editor) accumulateCount(key Key) bool {
	if (key >= '1' && key <= '9') || (key == '0' && e.Count > 0) {
		e.Count = e.Count*10 + int(key-'0')
		return true
	}
	return false
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
		e.handleNormal(b, key)
	case ModeInsert:
		e.handleInsert(b, key)
	case ModeVisual:
		e.handleVisual(b, key)
	}
}

func (e *Editor) handleNormal(b *Buffer, key Key) {
	if e.accumulateCount(key) {
		return
	}

	// "dd" and "cc"
	if key == 'd' || key == 'c' {
		if e.PendingOp == key {
			mult := e.multiplier()
			b.DeleteRows(b.Cy, mult)
			if key == 'c' {
				b.Rows = slices.Insert(b.Rows, b.Cy, []rune{})
				e.Mode = ModeInsert
			}
			e.clearState()
			return
		} else if e.PendingOp != 0 {
			// Invalid motions like "dc"
			e.clearState()
			return
		}
		e.PendingOp = key
		return
	}

	startMark := Mark{X: b.Cx, Y: b.Cy}
	// Vim has three types, charwise, linewise and blockwise
	// Charwise does things between two points, w, h, etc
	// Linewise doesn't care about Cx, just operates on full rows
	// Treats buffer like 2D grid, currently nothing uses it, probably absolute pain to implement
	isMotion, isLineWise := e.applyMotion(b, key)

	if isMotion {
		if e.PendingOp != 0 {
			e.applyPendingOp(b, startMark, isLineWise)
		}
		e.clearState()
		return
	}

	mult := e.multiplier()
	switch key {
	case 'x', KeyDelete:
		for range mult {
			b.DeleteChar(false)
		}
		e.clearState()
	case 'i':
		e.Mode = ModeInsert
		e.clearState()
	case 'v':
		e.Mode = ModeVisual
		b.Selection.Start = Mark{X: b.Cx, Y: b.Cy}
		b.Selection.Active = true
		e.clearState()
	case ctrlKey('s'):
		_ = b.Save()
	case ctrlKey('q'):
		e.Quit = true
	case ']':
		e.nextBuf()
	case '[':
		e.prevBuf()
	}
}

func (e *Editor) applyPendingOp(b *Buffer, start Mark, isLineWise bool) {
	endMark := Mark{X: b.Cx, Y: b.Cy}

	// Capture whole lines if linewise
	if isLineWise {
		if start.Y <= endMark.Y {
			start.X, endMark.X = 0, len(b.Rows[endMark.Y])
		} else {
			start.X, endMark.X = len(b.Rows[start.Y]), 0
		}
	}

	b.Selection.Start = start
	b.Selection.Active = true
	// Hacky!
	b.Cx, b.Cy = endMark.X, endMark.Y

	switch e.PendingOp {
	case 'd', 'c':
		b.DeleteSelection()
		if e.PendingOp == 'c' {
			e.Mode = ModeInsert
		}
	}
}

func (e *Editor) handleVisual(b *Buffer, key Key) {
	if e.accumulateCount(key) {
		return
	}

	isMotion, _ := e.applyMotion(b, key)
	if isMotion {
		e.clearState()
		return
	}

	switch key {
	case ctrlKey('q'):
		e.Quit = true
	case ctrlKey('s'):
		_ = b.Save()
	case 27, 'v':
		e.Mode = ModeNormal
		b.Selection.Clear()
	case 'd', 'x', 'c', KeyDelete:
		b.DeleteSelection()
		if key == 'c' {
			e.Mode = ModeInsert
		} else {
			e.Mode = ModeNormal
		}
	}
}

func (e *Editor) handleInsert(b *Buffer, key Key) {
	switch key {
	case ctrlKey('q'):
		e.Quit = true
	case ctrlKey('s'):
		_ = b.Save()
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
		b.MoveLeft()
	case KeyArrowDown:
		b.MoveDown()
	case KeyArrowUp:
		b.MoveUp()
	case KeyArrowRight:
		b.MoveRight()
	case KeyPageUp:
		_, termRows := e.render.Size()
		b.PageUp(termRows)
	case KeyPageDown:
		_, termRows := e.render.Size()
		b.PageDown(termRows)
	case KeyHome:
		b.MoveStart()
	case KeyEnd:
		b.MoveEnd()
	default:
		if unicode.IsPrint(rune(key)) {
			b.InsertChar(rune(key))
		}
	}
}

// Applies motions
// Returns: (isMotion, isLineWise)
func (e *Editor) applyMotion(b *Buffer, key Key) (bool, bool) {
	mult := e.multiplier()

	// "gg"
	if e.PendingMotion == 'g' {
		e.PendingMotion = 0
		if key == 'g' {
			b.Cy = 0
			if e.Count > 0 {
				b.Cy = max(0, min(e.Count-1, len(b.Rows)-1))
			}
			b.ClampCursor()
			return true, true
		}
		// Invalid like 'gj'
		return false, false
	}

	switch key {
	case 'g':
		// Wait for next key
		e.PendingMotion = 'g'
		return false, false
	case 'G':
		b.Cy = max(0, len(b.Rows)-1)
		if e.Count > 0 {
			b.Cy = max(0, min(e.Count-1, len(b.Rows)-1))
		}
		b.ClampCursor()
		return true, true
	case '0', KeyHome:
		b.MoveStart()
		return true, false
	case '$', KeyEnd:
		b.MoveEnd()
		return true, false
	case '^':
		b.MoveFirstNonBlank()
		return true, false
	case 'h', KeyArrowLeft:
		for range mult {
			b.MoveLeft()
		}
		return true, false
	case 'l', KeyArrowRight:
		for range mult {
			b.MoveRight()
		}
		return true, false
	case 'j', KeyArrowDown:
		for range mult {
			b.MoveDown()
		}
		return true, true
	case 'k', KeyArrowUp:
		for range mult {
			b.MoveUp()
		}
		return true, true
	case 'w':
		for range mult {
			b.MoveWord()
		}
		return true, false
	case 'b':
		for range mult {
			b.MoveBack()
		}
		return true, false
	case 'e':
		for range mult {
			b.MoveEndWord()
		}
		return true, false
	case KeyPageUp:
		_, termRows := e.render.Size()
		b.PageUp(termRows)
		return true, false
	case KeyPageDown:
		_, termRows := e.render.Size()
		b.PageDown(termRows)
		return true, false
	}
	return false, false
}
