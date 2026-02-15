package main

import (
	"bufio"
	"fmt"
	"os"

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
	Rows    [][]rune // The text buffer
	Cx, Cy  int      // Cursor pos
	RowOff  int      // Scroll offset row
	ColOff  int      // Scroll offser col
	Name    string   // Buffer name
	Dirty   bool     // Has the buffer been changed?
	Scratch bool     // Is this empty scrach buffer?
	Listed  bool     // Is this buffer listed?
}

type Editor struct {
	ScreenRows int // Terminal width
	ScreenCols int // Terminal height

	Mode       int      // Current mode
	CommandBuf string   // Command buffer
	Messages   []string // List of all messages
	MsgIndex   int      // Current message index
	Quit       bool     // Quitted?

	Buffers  []*Buffer // List of buffers
	BufIndex int       // Index of the active buffer
}

func main() {
	e := &Editor{
		Mode:    ModeNormal,
		Buffers: []*Buffer{},
	}
	e.addBuffer(true, false)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	e.updateWindowSize()

	reader := bufio.NewReader(os.Stdin)
	for !e.Quit {
		e.processKey(reader)
	}

	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
}

func (e *Editor) curBuf() *Buffer {
	if len(e.Buffers) == 0 {
		return nil
	}
	return e.Buffers[e.BufIndex]
}

func (e *Editor) addBuffer(listed bool, scratch bool) {
	name := ""
	if scratch {
		name = "Scratch"
	} else if listed {
		name = "No name"
	}
	buf := &Buffer{
		Rows:    [][]rune{{}},
		Name:    name,
		Scratch: scratch,
		Listed:  listed,
	}
	e.Buffers = append(e.Buffers, buf)
	e.BufIndex = len(e.Buffers) - 1
}

func (e *Editor) processKey(reader *bufio.Reader) {
	char, _, err := reader.ReadRune()
	if err != nil {
		return
	}
	if char == 'q' {
		e.Quit = true
	}
	fmt.Println(char)
}

func (e *Editor) updateWindowSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		e.ScreenCols = width
		e.ScreenRows = height - 1
	}
}
