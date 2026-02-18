package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/lukahietala/kiuru/internal/ansi"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	ModeNormal = iota
	ModeInsert
)

const (
	TabSize = 4
)

type Buffer struct {
	Rows      [][]rune // The text buffer
	Cx, Cy    int      // Real Cursor pos
	Vx, Vy    int      // Visual cursor position
	DesiredCx int      // "Sticky" cursor col memory (for snapping)
	RowOff    int      // Scroll offset row
	ColOff    int      // Scroll offset col
	Name      string   // Buffer name
	Dirty     bool     // Has the buffer been changed?
	Scratch   bool     // Is this empty scratch buffer?
	Listed    bool     // Is this buffer listed?
}

type Editor struct {
	mu sync.Mutex

	TermRows int // Terminal height
	TermCols int // Terminal width

	Mode       int    // Current mode
	CommandBuf string // Command buffer
	Message    string // Current message
	Quit       bool   // Quitted?

	Buffers  []*Buffer // List of buffers
	BufIndex int       // Index of the active buffer

	renderBuf bytes.Buffer

	Debug bool
}

func main() {
	debugPtr := flag.Bool("debug", false, "show runtime stats")
	flag.Parse()

	e := &Editor{
		Mode:    ModeNormal,
		Buffers: []*Buffer{},
		Debug:   *debugPtr,
	}

	args := flag.Args()
	if len(args) > 0 {
		for _, arg := range args {
			e.openFile(arg)
		}
		e.BufIndex = 0
	} else {
		e.addBuffer(true, false)
	}

	// Setup terminal
	os.Stdout.WriteString(ansi.AltBufferOn + ansi.ClearScreen + ansi.CursorHome)
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		os.Stdout.WriteString(ansi.AltBufferOff + ansi.ClearScreen + ansi.CursorHome)
	}()

	e.updateWindowSize()

	// Listen for SIGWINCH to know when to resize terminal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		for range sigChan {
			e.mu.Lock()
			e.updateWindowSize()
			e.render()
			e.mu.Unlock()
		}
	}()

	// TODO: Handle SIGHUP
	// TODO: Recover from panic, save, then exit

	reader := bufio.NewReader(os.Stdin)

	for {
		e.mu.Lock()
		if e.Quit {
			e.mu.Unlock()
			break
		}
		e.render()
		e.mu.Unlock()

		char, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		e.mu.Lock()
		e.processKey(char)
		e.mu.Unlock()
	}
}

// Get current buffer
func (e *Editor) curBuf() *Buffer {
	if len(e.Buffers) == 0 {
		return nil
	}
	return e.Buffers[e.BufIndex]
}

// Opens file and create a buffer for it
func (e *Editor) openFile(path string) {
	rows := [][]rune{}

	file, err := os.Open(path)
	if err != nil {
		rows = append(rows, []rune{})
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			rows = append(rows, []rune(line))
		}
		if len(rows) == 0 {
			rows = append(rows, []rune{})
		}
	}
	buf := &Buffer{
		Rows:      rows,
		Name:      path,
		Scratch:   false,
		Listed:    true,
		Dirty:     false,
		DesiredCx: 0,
	}
	e.Buffers = append(e.Buffers, buf)
	e.BufIndex = len(e.Buffers) - 1
}

// Creates an empty buffer
func (e *Editor) addBuffer(listed bool, scratch bool) {
	name := ""
	if scratch {
		name = "Scratch"
	} else if listed {
		name = "No name"
	}
	buf := &Buffer{
		Rows:      [][]rune{{}},
		Name:      name,
		Scratch:   scratch,
		Listed:    listed,
		DesiredCx: 0,
	}
	e.Buffers = append(e.Buffers, buf)
	e.BufIndex = len(e.Buffers) - 1
}

func (e *Editor) insertChar(char rune) {
	b := e.curBuf()

	for b.Cy >= len(b.Rows) {
		b.Rows = append(b.Rows, []rune{})
	}

	b.Rows[b.Cy] = slices.Insert(b.Rows[b.Cy], b.Cx, char)

	b.Cx++
	b.Dirty = true
	b.DesiredCx = b.Cx
}

func (e *Editor) insertNewline() {
	b := e.curBuf()

	for b.Cy >= len(b.Rows) {
		b.Rows = append(b.Rows, []rune{})
	}

	currentRow := b.Rows[b.Cy]

	// Save chars from right
	remainder := make([]rune, len(currentRow[b.Cx:]))
	copy(remainder, currentRow[b.Cx:])

	// Remove chars from right
	b.Rows[b.Cy] = currentRow[:b.Cx]

	// Insert the remainder as a new row
	b.Rows = slices.Insert(b.Rows, b.Cy+1, remainder)

	b.Cy++
	b.Cx = 0
	b.Dirty = true
	b.DesiredCx = 0
}

func (e *Editor) deleteChar() {
	b := e.curBuf()

	if b.Cx > 0 {
		// Middle of the line
		b.Rows[b.Cy] = slices.Delete(b.Rows[b.Cy], b.Cx-1, b.Cx)
		b.Cx--
	} else if b.Cy > 0 {
		// Start of line (merge with previous)
		prevRowIndex := b.Cy - 1
		// Move cursor to previous end
		b.Cx = len(b.Rows[prevRowIndex])

		// Merge current row into previous
		b.Rows[prevRowIndex] = append(b.Rows[prevRowIndex], b.Rows[b.Cy]...)

		// Remove the empty current row
		b.Rows = slices.Delete(b.Rows, b.Cy, b.Cy+1)
		b.Cy--
	} else {
		return
	}

	b.Dirty = true
	b.DesiredCx = b.Cx
}

// Handles all keypresses
func (e *Editor) processKey(char rune) {
	b := e.curBuf()
	if b == nil {
		return
	}

	switch e.Mode {
	case ModeNormal:
		switch char {
		case 'q':
			e.Quit = true
		case 'i':
			e.Mode = ModeInsert
		case 'h':
			if b.Cx > 0 {
				b.Cx--
				b.DesiredCx = b.Cx
			}
		case 'j':
			if b.Cy < len(b.Rows)-1 {
				b.Cy++
				e.clampCursor(b)
			}
		case 'k':
			if b.Cy > 0 {
				b.Cy--
				e.clampCursor(b)
			}
		case 'l':
			if b.Cy < len(b.Rows) && b.Cx < len(b.Rows[b.Cy]) {
				b.Cx++
				b.DesiredCx = b.Cx
			}
		}
	case ModeInsert:
		switch char {
		case 27: // Escape key
			e.Mode = ModeNormal
		case 13: // Enter
			e.insertNewline()
		case 127: // Backspace
			e.deleteChar()
		case '\t': // Tab
			e.insertChar('\t')
		default:
			if unicode.IsPrint(char) {
				e.insertChar(char)
			}
		}
	}
}

// Snaps the cursor to the end of shorter lines while remembering
// the original column (DesiredCx) so it can restore it on longer lines
func (e *Editor) clampCursor(b *Buffer) {
	rowLen := 0
	if b.Cy < len(b.Rows) {
		rowLen = len(b.Rows[b.Cy])
	}
	b.Cx = min(b.DesiredCx, rowLen)
}

// Determines line gutter width
func (e *Editor) getGutterWidth(b *Buffer) int {
	digits := len(fmt.Sprintf("%d", len(b.Rows)))
	digits = max(3, digits)
	return digits + 2
}

// "Scrolls" the view to right place for renderer
func (e *Editor) scroll() {
	b := e.curBuf()
	if b == nil {
		return
	}

	// Vertical scrolling
	if b.Cy < b.RowOff {
		b.RowOff = b.Cy
	}
	// TODO: Scrolloff
	if b.Cy >= b.RowOff+e.TermRows {
		b.RowOff = b.Cy - e.TermRows + 1
	}

	// Calculate vx based on tabss
	b.Vx = 0
	if b.Cy < len(b.Rows) {
		row := b.Rows[b.Cy]
		for i := 0; i < b.Cx && i < len(row); i++ {
			char := row[i]
			if char == '\t' {
				b.Vx += (TabSize - 1) - (b.Vx % TabSize)
				b.Vx++
			} else {
				b.Vx += runewidth.RuneWidth(char)
			}
		}
	}

	// Horizontal scrolling
	// Reduce terminal width by the size of the gutter
	gutterWidth := e.getGutterWidth(b)
	termWidth := e.TermCols - gutterWidth

	if b.Vx < b.ColOff {
		b.ColOff = b.Vx
	}
	if b.Vx >= b.ColOff+termWidth {
		b.ColOff = b.Vx - termWidth + 1
	}
}

// Draws the screen
func (e *Editor) render() {
	start := time.Now()
	e.renderBuf.Reset()
	e.renderBuf.WriteString(ansi.HideCursor + ansi.CursorHome)

	b := e.curBuf()
	e.scroll()

	gutterWidth := e.getGutterWidth(b)
	textWidth := e.TermCols - gutterWidth

	for y := range e.TermRows {
		bufRow := y + b.RowOff

		// Draw line gutter
		if bufRow < len(b.Rows) {
			e.renderBuf.WriteString(ansi.DimMode)
			// Right align line number, leave some padding
			fmt.Fprintf(&e.renderBuf, "%*d ", gutterWidth-1, bufRow+1)
			e.renderBuf.WriteString(ansi.ResetFormat)
		}

		// Draw buffer rows
		if bufRow >= len(b.Rows) {
			e.renderBuf.WriteString(ansi.DimMode)
			e.renderBuf.WriteString("~")
			e.renderBuf.WriteString(ansi.ResetFormat)
		} else {
			line := b.Rows[bufRow]
			// Visual char x pos
			rx := 0

			for _, c := range line {
				w := runewidth.RuneWidth(c)
				// Get real tab width
				if c == '\t' {
					w = TabSize - (rx % TabSize)
				}

				// If character starts past the right edge, stop
				if rx-b.ColOff >= textWidth {
					break
				}

				// If character ends past right edge, stop
				if (rx-b.ColOff)+w > textWidth {
					break
				}

				// If visible
				if rx >= b.ColOff {
					if c == '\t' {
						// Render tabs
						for i := 0; i < w; i++ {
							e.renderBuf.WriteByte(' ')
						}
					} else {
						e.renderBuf.WriteRune(c)
					}
				}
				rx += w
			}
		}

		// Clear line to prevent ghosts
		e.renderBuf.WriteString(ansi.ClearLine)
		if y < e.TermRows-1 {
			e.renderBuf.WriteString("\r\n")
		}
	}

	// Start rendering status bar
	e.renderBuf.WriteString(ansi.MoveCursor(e.TermRows+1, 1))
	e.renderBuf.WriteString(ansi.ReverseVideo)

	// TODO!: clean up
	// Left side, basic info
	statusLeft := fmt.Sprintf(" %s - (%d,%d)", b.Name, b.Cy+1, b.Cx+1)

	// Right side, debug
	statusRight := ""
	if e.Debug {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		totalPause := time.Duration(m.PauseTotalNs)
		statusRight = fmt.Sprintf("Render:%dµs Obj:%dk Pauses:%v Alloc:%vMB Sys:%vMB GC:%v",
			time.Since(start).Microseconds(),
			m.HeapObjects/1000,
			totalPause,
			m.Alloc/1024/1024,
			m.Sys/1024/1024,
			m.NumGC,
		)
	}

	// Add padding between right and left sides
	padding := e.TermCols - len(statusLeft)
	if e.Debug {
		padding -= len(statusRight)
	}

	if padding < 0 {
		// Truncate left status if no room (less important)
		if len(statusLeft) > e.TermCols {
			statusLeft = statusLeft[:e.TermCols]
		}
		e.renderBuf.WriteString(statusLeft)
	} else {
		e.renderBuf.WriteString(statusLeft)
		for i := 0; i < padding; i++ {
			e.renderBuf.WriteByte(' ')
		}
		if e.Debug {
			e.renderBuf.WriteString(statusRight)
		}
	}

	e.renderBuf.WriteString(ansi.ResetFormat)
	// Position cursor
	screenY := (b.Cy - b.RowOff) + 1
	screenX := (b.Vx - b.ColOff) + 1 + gutterWidth

	// Make sure that cursor is in view
	if screenY >= 1 && screenY <= e.TermRows {
		e.renderBuf.WriteString(ansi.MoveCursor(screenY, screenX))
	}
	// Show cursor
	e.renderBuf.WriteString(ansi.ShowCursor)
	os.Stdout.Write(e.renderBuf.Bytes())
}

// Updates editor window size and keeps cursor clamped
func (e *Editor) updateWindowSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		e.TermCols = width
		// Reserve 1 line for status bar
		e.TermRows = height - 1

		// Keep cursor in bounds
		b := e.curBuf()
		if b != nil {
			if b.Cy >= len(b.Rows) {
				b.Cy = len(b.Rows) - 1
			}
			e.clampCursor(b)
		}
	}
}
