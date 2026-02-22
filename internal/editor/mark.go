package editor

// Position in buffer
type Mark struct {
	X, Y int
}

type SelectionRange struct {
	Start  Mark
	Active bool
}

// Returns the start and end marks in top-to-bottom, left-to-right order
func (s *SelectionRange) NormalizedBounds(cursorX, cursorY int) (start Mark, end Mark) {
	endMark := Mark{X: cursorX, Y: cursorY}

	if s.Start.Y < endMark.Y || (s.Start.Y == endMark.Y && s.Start.X <= endMark.X) {
		return s.Start, endMark
	}
	return endMark, s.Start
}

// Check if x, y coords are in selection
func (s *SelectionRange) Contains(x, y, cursorX, cursorY int) bool {
	if !s.Active {
		return false
	}

	start, end := s.NormalizedBounds(cursorX, cursorY)

	// Outside vertical bounds
	if y < start.Y || y > end.Y {
		return false
	}
	// Single line selection
	if y == start.Y && y == end.Y {
		return x >= start.X && x <= end.X
	}
	// Multi line selection, top line
	if y == start.Y {
		return x >= start.X
	}
	// Multi line selection, bottom line
	if y == end.Y {
		return x <= end.X
	}
	// Middle lines
	return true
}

// Clears active visual selection
func (s *SelectionRange) Clear() {
	s.Active = false
}
