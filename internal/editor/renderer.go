package editor

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lukahietala/kiuru/internal/ansi"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type Renderer interface {
	Render(b *Buffer, mode Mode, debug bool)
	UpdateSize() (cols, rows int, err error)
	Size() (cols, rows int)
}

type TermRenderer struct {
	out  io.Writer // Usually stdout
	fd   int
	cols int
	rows int
	buf  bytes.Buffer // Render buffer, flushed once ready
}

func NewTermRenderer(out io.Writer, fd int) *TermRenderer {
	return &TermRenderer{out: out, fd: fd}
}

func (r *TermRenderer) UpdateSize() (int, int, error) {
	w, h, err := term.GetSize(r.fd)
	if err == nil {
		r.cols = w
		r.rows = h - 1 // Reserve 1 line for statusbar
	}
	return r.cols, r.rows, err
}

func (r *TermRenderer) Size() (int, int) {
	return r.cols, r.rows
}

func getGutterWidth(b *Buffer) int {
	digits := len(fmt.Sprintf("%d", len(b.Rows)))
	return max(3, digits) + 2
}

func (r *TermRenderer) Render(b *Buffer, mode Mode, debug bool) {
	start := time.Now()
	r.buf.Reset()
	r.buf.WriteString(ansi.HideCursor + ansi.CursorHome)

	if b == nil {
		return
	}

	gutterWidth := getGutterWidth(b)
	textWidth := r.cols - gutterWidth

	for y := 0; y < r.rows; y++ {
		bufRow := y + b.RowOff

		if bufRow >= len(b.Rows) {
			r.drawEmptyLine()
		} else {
			r.drawGutter(gutterWidth, bufRow)
			r.drawLine(b, bufRow, textWidth, mode)
		}

		r.buf.WriteString(ansi.ClearLine)
		if y < r.rows-1 {
			r.buf.WriteString("\r\n")
		}
	}

	r.renderStatusBar(b, mode, debug, start)

	screenY := (b.Cy - b.RowOff) + 1
	screenX := (b.Vx - b.ColOff) + 1 + gutterWidth
	if screenY >= 1 && screenY <= r.rows {
		r.buf.WriteString(ansi.MoveCursor(screenY, screenX))
	}

	r.buf.WriteString(ansi.ShowCursor)
	r.out.Write(r.buf.Bytes())
}

func (r *TermRenderer) drawGutter(width, row int) {
	r.buf.WriteString(ansi.DimMode)
	fmt.Fprintf(&r.buf, "%*d ", width-1, row+1)
	r.buf.WriteString(ansi.ResetFormat)
}

func (r *TermRenderer) drawEmptyLine() {
	r.buf.WriteString(ansi.DimMode + "~" + ansi.ResetFormat)
}

func (r *TermRenderer) drawLine(b *Buffer, bufRow, textWidth int, mode Mode) {
	line := b.Rows[bufRow]
	rx := 0

	for i, c := range line {
		renderStr, charWidth, isDim := formatRune(c, rx)

		if rx-b.ColOff >= textWidth || (rx-b.ColOff)+charWidth > textWidth {
			break
		}

		if rx >= b.ColOff {
			isSelected := mode == ModeVisual && b.Selection.Contains(i, bufRow, b.Cx, b.Cy)

			if isSelected {
				r.buf.WriteString(ansi.ReverseVideo)
			}
			if isDim {
				r.buf.WriteString(ansi.DimMode)
			}

			r.buf.WriteString(renderStr)

			if isDim || isSelected {
				r.buf.WriteString(ansi.ResetFormat)
			}
		}
		rx += charWidth
	}
}

// Determine how should a char be rendered
func formatRune(c rune, rx int) (renderStr string, charWidth int, isDim bool) {
	// Tabs
	if c == '\t' {
		charWidth = TabSize - (rx % TabSize)
		return fmt.Sprintf("%*s", charWidth, ""), charWidth, false
	}

	if c < 32 || c == 127 {
		// Magically maps \f to L, \r to M, DEL to ?... etc (caret notation)
		// Similar to vim's transchar_nonprint()
		return "^" + string(byte(c)+'@'), 2, true
	}

	// Invalid utf
	if c == utf8.RuneError {
		return "U+FFFD", 6, true
	}

	// TODO: Messy, try to find a way to determine if binary and use hex mode like nvim
	if !unicode.IsGraphic(c) {
		if c <= 0xFF {
			// Single byte (ascii)
			renderStr = fmt.Sprintf("<%02X>", c)
		} else {
			// Over one byte (unicode)
			renderStr = fmt.Sprintf("<%04X>", c)
		}
		return renderStr, len(renderStr), true
	}

	// Normal characters
	return string(c), runewidth.RuneWidth(c), false
}

func (r *TermRenderer) renderStatusBar(b *Buffer, mode Mode, debug bool, start time.Time) {
	r.buf.WriteString(ansi.MoveCursor(r.rows+1, 1) + ansi.ReverseVideo)

	modeStr := "normal"
	switch mode {
	case ModeInsert:
		modeStr = "insert"
	case ModeVisual:
		modeStr = "visual"
	}

	readOnlyStr := ""
	if b.ReadOnly {
		readOnlyStr = "(RO) "
	}

	statusLeft := fmt.Sprintf("%s%s - (%d,%d) (%s)", readOnlyStr, b.Name, b.Cy+1, b.Cx+1, modeStr)

	statusRight := ""
	if debug {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		statusRight = fmt.Sprintf("Render:%dµs Alloc:%vMB Sys:%vMB",
			time.Since(start).Microseconds(), m.Alloc/1024/1024, m.Sys/1024/1024)
	}

	padding := r.cols - len(statusLeft)
	if debug {
		padding -= len(statusRight)
	}

	if padding < 0 {
		if len(statusLeft) > r.cols {
			statusLeft = statusLeft[:r.cols]
		}
		r.buf.WriteString(statusLeft)
	} else {
		r.buf.WriteString(statusLeft)
		for i := 0; i < padding; i++ {
			r.buf.WriteByte(' ')
		}
		if debug {
			r.buf.WriteString(statusRight)
		}
	}
	r.buf.WriteString(ansi.ResetFormat)
}
