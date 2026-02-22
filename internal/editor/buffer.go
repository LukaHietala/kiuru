package editor

import (
	"bufio"
	"errors"
	"os"
	"strings"
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
	Selection SelectionRange
	Format    FileFormat
}

// Creates a new buffer
// TODO: Move file logic
func NewBuffer(path string, scratch, listed bool) (*Buffer, error) {
	b := &Buffer{
		Name:    path,
		Scratch: scratch,
		Listed:  listed,
		Rows:    [][]rune{},
		Format:  FormatUnix,
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

	reader := bufio.NewReader(file)
	var rawLines []string
	// Try to smartly detect line endings
	// TODO: Change?
	dosCount, unixCount := 0, 0

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if strings.HasSuffix(line, "\r\n") {
				dosCount++
			} else if strings.HasSuffix(line, "\n") {
				unixCount++
			}
			rawLines = append(rawLines, line)
		}
		if err != nil {
			break
		}
	}

	if dosCount > unixCount {
		b.Format = FormatDOS
	}

	for _, line := range rawLines {
		line = strings.TrimSuffix(line, "\n")
		// If DOS strip \r too, if UNIX leave it there so it can be rendered as ^M
		if b.Format == FormatDOS {
			line = strings.TrimSuffix(line, "\r")
		}
		b.Rows = append(b.Rows, []rune(line))
	}

	if len(b.Rows) == 0 {
		b.Rows = append(b.Rows, []rune{})
	}

	return b, nil
}
