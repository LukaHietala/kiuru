package editor

import (
	"bufio"
	"io"
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

	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
	// TODO: Add all
	if char == '\x1b' {
		if t.reader.Buffered() == 0 {
			return Key(char), nil
		}
		seq1, _, err := t.reader.ReadRune()
		if err != nil {
			return Key(char), nil
		}

		if seq1 == '[' {
			seq2, _, err := t.reader.ReadRune()
			if err != nil {
				return Key(char), nil
			}

			if seq2 >= '0' && seq2 <= '6' {
				seq3, _, _ := t.reader.ReadRune()
				if seq3 == '~' {
					switch seq2 {
					case '1':
						return KeyHome, nil
					case '3':
						return KeyDelete, nil
					case '4':
						return KeyEnd, nil
					case '5':
						return KeyPageUp, nil
					case '6':
						return KeyPageDown, nil
					}
				}
			} else {
				switch seq2 {
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
				}
			}
		}
	}
	return Key(char), nil
}
