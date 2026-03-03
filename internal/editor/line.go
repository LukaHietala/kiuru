package editor

import "slices"

// Inserts rune to cursor pos
func (b *Buffer) InsertChar(char rune) {
	for b.Cy >= len(b.Rows) {
		b.Rows = append(b.Rows, []rune{})
	}
	b.Rows[b.Cy] = slices.Insert(b.Rows[b.Cy], b.Cx, char)
	b.Cx++
	b.Dirty = true
	b.DesiredCx = b.Cx
}

// Appends new row below cursor, handles line spilts
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

// Deletes rune next or before cursor
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

// Deletes everything between selection start and end
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
		// Single line delete
		if start.X <= end.X && start.X < len(b.Rows[start.Y]) {
			b.Rows[start.Y] = slices.Delete(b.Rows[start.Y], start.X, end.X+1)
		}
	} else {
		// Multi line delete
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

// Clamps cursor to line and tries to move to desired place (DesiredCx)
func (b *Buffer) ClampCursor() {
	rowLen := 0
	if b.Cy < len(b.Rows) {
		rowLen = len(b.Rows[b.Cy])
	}
	b.Cx = min(b.DesiredCx, rowLen)
}

func (b *Buffer) DeleteRows(start, count int) {
	if len(b.Rows) == 0 || start >= len(b.Rows) {
		return
	}

	end := min(start+count, len(b.Rows))
	b.Rows = slices.Delete(b.Rows, start, end)

	// Keep at least one empty row
	if len(b.Rows) == 0 {
		b.Rows = [][]rune{{}}
	}

	// Adjust cursor
	if b.Cy >= len(b.Rows) {
		b.Cy = len(b.Rows) - 1
	}
	b.Cx = 0
	b.DesiredCx = 0
	b.Dirty = true
}
