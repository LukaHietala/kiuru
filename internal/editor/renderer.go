package editor

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"time"

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
	// Usually stdout
	out  io.Writer
	fd   int
	cols int
	rows int
	// Render buffer, will be flused once ready
	buf bytes.Buffer
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
	// TODO: Handle control seqs, non-printable, broken utf and binary
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

		if bufRow < len(b.Rows) {
			r.buf.WriteString(ansi.DimMode)
			fmt.Fprintf(&r.buf, "%*d ", gutterWidth-1, bufRow+1)
			r.buf.WriteString(ansi.ResetFormat)
		}

		if bufRow >= len(b.Rows) {
			r.buf.WriteString(ansi.DimMode + "~" + ansi.ResetFormat)
		} else {
			line := b.Rows[bufRow]
			rx := 0
			for _, c := range line {
				// Get real width since some chars are 0-2 cols wide
				w := runewidth.RuneWidth(c)
				if c == '\t' {
					w = TabSize - (rx % TabSize)
				}
				if rx-b.ColOff >= textWidth || (rx-b.ColOff)+w > textWidth {
					break
				}
				if rx >= b.ColOff {
					if c == '\t' {
						fmt.Fprintf(&r.buf, "%*s", w, "")
					} else {
						r.buf.WriteRune(c)
					}
				}
				rx += w
			}
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

func (r *TermRenderer) renderStatusBar(b *Buffer, mode Mode, debug bool, start time.Time) {
	r.buf.WriteString(ansi.MoveCursor(r.rows+1, 1) + ansi.ReverseVideo)

	modeStr := "normal"
	if mode == ModeInsert {
		modeStr = "insert"
	}

	readOnlyStr := ""
	if b.ReadOnly {
		readOnlyStr = "(RO)"
	}

	statusLeft := fmt.Sprintf("%s %s - (%d,%d) (%s)", readOnlyStr, b.Name, b.Cy+1, b.Cx+1, modeStr)

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
