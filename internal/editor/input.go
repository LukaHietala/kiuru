package editor

import (
	"bufio"
	"io"
	"time"
)

type Key rune

const (
	KeyArrowUp = 1000 + iota
	KeyArrowDown
	KeyArrowRight
	KeyArrowLeft
	KeyInsert
	KeyDelete
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
)

type InputReader interface {
	ReadKey() (Key, error)
	Buffered() int
}

type TermInput struct {
	reader *bufio.Reader
}

func NewTermInput(r io.Reader) *TermInput {
	return &TermInput{reader: bufio.NewReader(r)}
}

func (t *TermInput) Buffered() int {
	return t.reader.Buffered()
}

func (t *TermInput) ReadKey() (Key, error) {
	char, _, err := t.reader.ReadRune()
	if err != nil {
		return 0, err
	}

	// Not escape, so regular char
	if char != '\x1b' {
		return Key(char), nil
	}

	// On SSH/Slow terminals next seq after escape might take some time
	// So we wait some time for it, if nothing comes then assume its regular escape
	// "ttimeout" in Vim
	// TODO: Make configurable (currently vim default "ttimeoutlen")
	if !t.waitForBytes(50 * time.Millisecond) {
		return Key(char), nil
	}

	// Peek at the next character
	seq1, _, err := t.reader.ReadRune()
	if err != nil {
		return Key(char), nil
	}

	// If there is '[', treat it as CSI
	if seq1 == '[' {
		return t.handleCSI()
	}

	// Otherwise, it's likely an alt something
	// Currently just return as regular char, maybe mark it as alt char later, TODO?
	return Key(seq1), nil
}

// handleCSI deals with sequences starting with '\x1b['
// Bible: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// To debug: showkey -a
func (t *TermInput) handleCSI() (Key, error) {
	char, _, err := t.reader.ReadRune()
	if err != nil {
		return 0, err
	}

	switch char {
	case 'A':
		return KeyArrowUp, nil
	case 'B':
		return KeyArrowDown, nil
	case 'C':
		return KeyArrowRight, nil
	case 'D':
		return KeyArrowLeft, nil
	case 'H':
		return KeyHome, nil
	case 'F':
		return KeyEnd, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// These are VT sequences that usually end in '~' (like [6~ <PageDown>)
		return t.parseVT(char)
	}

	return 0, nil
}

// Handles VT seqs
func (t *TermInput) parseVT(firstDigit rune) (Key, error) {
	next, _, err := t.reader.ReadRune()
	if err != nil {
		return 0, err
	}

	// TODO: Handle two digit ones too
	if next != '~' {
		return 0, nil
	}

	switch firstDigit {
	case '1', '7':
		return KeyHome, nil
	case '3':
		return KeyDelete, nil
	case '4', '8':
		return KeyEnd, nil
	case '5':
		return KeyPageUp, nil
	case '6':
		return KeyPageDown, nil
	}

	return 0, nil
}

// TODO??: Channels to make non blocking?
func (t *TermInput) waitForBytes(timeout time.Duration) bool {
	if t.reader.Buffered() > 0 {
		return true
	}
	start := time.Now()
	for time.Since(start) < timeout {
		if t.reader.Buffered() > 0 {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}
