package editor

import (
	"bufio"
	"errors"
	"os"
	"slices"
)

type Buffer struct {
	Rows      [][]rune
	Cx, Cy    int
	Vx, Vy    int
	DesiredCx int
	RowOff    int
	ColOff    int
	Name      string
	Dirty     bool
	Scratch   bool
	Listed    bool
	ReadOnly  bool
	Selection VisualSelection
}

func NewBuffer(path string, scratch, listed bool) (*Buffer, error) {
	b := &Buffer{
		Name:    path,
		Scratch: scratch,
		Listed:  listed,
		Rows:    [][]rune{},
	}

	if scratch {
		b.Name = "Scratch"
		b.Rows = [][]rune{{}}
		return b, nil
	}

	if path == "" {
		b.Name = "No name"
		b.Rows = [][]rune{{}}
		return b, nil
	}

	fileInfo, err := os.Stat(path)
	if err == nil {
		// Can open in write mode?
		f, err := os.OpenFile(path, os.O_WRONLY, 0666)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				b.ReadOnly = true
			}
		} else {
			f.Close()
		}

		// Marked as readonly?
		if fileInfo.Mode().Perm()&0200 == 0 {
			b.ReadOnly = true
		}
	}

	file, err := os.Open(path)
	if err != nil {
		b.Rows = append(b.Rows, []rune{})
		return b, err
	}
	defer file.Close()

	// TODO: Use reader and detect line endings
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		b.Rows = append(b.Rows, []rune(scanner.Text()))
	}
	if len(b.Rows) == 0 {
		b.Rows = append(b.Rows, []rune{})
	}
	return b, nil
}

func (b *Buffer) InsertChar(char rune) {
	for b.Cy >= len(b.Rows) {
		b.Rows = append(b.Rows, []rune{})
	}
	b.Rows[b.Cy] = slices.Insert(b.Rows[b.Cy], b.Cx, char)
	b.Cx++
	b.Dirty = true
	b.DesiredCx = b.Cx
}

func (b *Buffer) InsertNewline() {
	for b.Cy >= len(b.Rows) {
		b.Rows = append(b.Rows, []rune{})
	}

	curRow := b.Rows[b.Cy]
	remainder := make([]rune, len(curRow[b.Cx:]))
	copy(remainder, curRow[b.Cx:])

	b.Rows[b.Cy] = curRow[:b.Cx]
	b.Rows = slices.Insert(b.Rows, b.Cy+1, remainder)

	b.Cy++
	b.Cx = 0
	b.Dirty = true
	b.DesiredCx = 0
}

func (b *Buffer) DeleteChar(backspace bool) {
	if backspace {
		if b.Cx > 0 {
			b.Rows[b.Cy] = slices.Delete(b.Rows[b.Cy], b.Cx-1, b.Cx)
			b.Cx--
		} else if b.Cy > 0 {
			prevRowIndex := b.Cy - 1
			b.Cx = len(b.Rows[prevRowIndex])
			b.Rows[prevRowIndex] = append(b.Rows[prevRowIndex], b.Rows[b.Cy]...)
			b.Rows = slices.Delete(b.Rows, b.Cy, b.Cy+1)
			b.Cy--
		} else {
			return
		}
	} else { // DEL or x
		if b.Cx < len(b.Rows[b.Cy]) {
			b.Rows[b.Cy] = slices.Delete(b.Rows[b.Cy], b.Cx, b.Cx+1)
		} else if b.Cy < len(b.Rows)-1 {
			nextRowIndex := b.Cy + 1
			b.Rows[b.Cy] = append(b.Rows[b.Cy], b.Rows[nextRowIndex]...)
			b.Rows = slices.Delete(b.Rows, nextRowIndex, nextRowIndex+1)
		} else {
			return
		}
	}
	b.Dirty = true
	b.DesiredCx = b.Cx
}

func (b *Buffer) DeleteSelection() {
	if !b.Selection.Active {
		return
	}
	start, end := b.Selection.NormalizedBounds(b.Cx, b.Cy)

	if start.Y >= len(b.Rows) {
		return
	}
	if end.Y >= len(b.Rows) {
		end.Y = len(b.Rows) - 1
	}

	startRowLen := len(b.Rows[start.Y])
	if start.X > startRowLen {
		start.X = startRowLen
	}

	endRowLen := len(b.Rows[end.Y])
	if end.X >= endRowLen {
		end.X = max(0, endRowLen-1)
	}

	if start.Y == end.Y {
		// Single-line delete
		if start.X <= end.X && start.X < len(b.Rows[start.Y]) {
			b.Rows[start.Y] = slices.Delete(b.Rows[start.Y], start.X, end.X+1)
		}
	} else {
		// Multi-line delete
		firstHalf := b.Rows[start.Y][:start.X]

		var secondHalf []rune
		if end.X+1 < len(b.Rows[end.Y]) {
			secondHalf = b.Rows[end.Y][end.X+1:]
		}

		b.Rows[start.Y] = append(firstHalf, secondHalf...)
		b.Rows = slices.Delete(b.Rows, start.Y+1, end.Y+1)
	}

	b.Cx = start.X
	b.Cy = start.Y
	b.DesiredCx = b.Cx
	b.ClampCursor()
	b.Selection.Clear()
	b.Dirty = true
}

func (b *Buffer) ClampCursor() {
	rowLen := 0
	if b.Cy < len(b.Rows) {
		rowLen = len(b.Rows[b.Cy])
	}
	b.Cx = min(b.DesiredCx, rowLen)
}
