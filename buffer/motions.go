package buffer

func (b *Buffer) MoveLeft() {
	b.cursorX -= PrevRuneSize(b.lines[b.cursorY], b.cursorX)
}

func (b *Buffer) MoveRight() {
	b.cursorX += NextRuneSize(b.lines[b.cursorY], b.cursorX)
}

func (b *Buffer) MoveUp() {
	if b.cursorY > 0 {
		startLine := b.lines[b.cursorY]
		endLine := b.lines[b.cursorY-1]
		b.cursorX = AlignOffsetByRune(startLine, endLine, b.cursorX)
		b.cursorY--
	}
}

func (b *Buffer) MoveDown() {
	if b.cursorY < len(b.lines)-1 {
		startLine := b.lines[b.cursorY]
		endLine := b.lines[b.cursorY+1]
		b.cursorX = AlignOffsetByRune(startLine, endLine, b.cursorX)
		b.cursorY++
	}
}

func (b *Buffer) ClampCursor() {
	// TODO: Desired x
	b.cursorY = max(0, min(b.cursorY, len(b.lines)-1))
	b.cursorX = max(0, min(b.cursorX, len(b.lines[b.cursorY])))
}
