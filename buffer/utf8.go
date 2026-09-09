package buffer

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// TODO: Render rune error (RuneError = '\uFFFD')

// PrevRuneSize returns the size of previous rune (left)
func PrevRuneSize(line []byte, offset int) int {
	if offset <= 0 || offset > len(line) {
		return 0
	}
	_, size := utf8.DecodeLastRune(line[:offset])
	return size
}

// NextRuneSize returns the size of next rune (right)
func NextRuneSize(line []byte, offset int) int {
	if offset < 0 || offset >= len(line) {
		return 0
	}
	_, size := utf8.DecodeRune(line[offset:])
	return size
}

// ByteToRuneIndex converts a byte offset to a rune index
func ByteToRuneIndex(line []byte, offset int) int {
	offset = max(0, min(offset, len(line)))
	return utf8.RuneCount(line[:offset])
}

// RuneIndexToByteOffset maps a rune index back to its starting byte pos
func RuneIndexToByteOffset(line []byte, runeIndex int) int {
	if runeIndex <= 0 {
		return 0
	}

	offset := 0
	for i := 0; i < runeIndex && offset < len(line); i++ {
		_, size := utf8.DecodeRune(line[offset:])
		offset += size
	}
	return offset
}

// AlignOffsetByRune maps a byte offset from startLine to endLine by character index
func AlignOffsetByRune(startLine, endLine []byte, offset int) int {
	return RuneIndexToByteOffset(endLine, ByteToRuneIndex(startLine, offset))
}

// VisualWidth calculates display width of runes to offset
func VisualWidth(line []byte, offset int) int {
	offset = max(0, min(offset, len(line)))
	width := 0
	for i := 0; i < offset; {
		r, size := utf8.DecodeRune(line[i:offset])
		width += runewidth.RuneWidth(r)
		i += size
	}
	return width
}

// TODO: Visual to byte  offset and tabs :((((((
//aksldjfa flk
