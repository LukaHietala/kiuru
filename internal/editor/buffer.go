package editor

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
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
	AbsPath   string
	Dirty     bool
	Scratch   bool
	Listed    bool
	ReadOnly  bool
	Selection SelectionRange
	Format    FileFormat
}

// Creates a new buffer
func NewBuffer(path string, scratch, listed bool) (*Buffer, error) {
	// Try to resolve abs path
	absPath, err := filepath.Abs(path)
	if err != nil {
		// TODO!
	}

	if path == "" {
		absPath = ""
	}

	b := &Buffer{
		Name:    path,
		AbsPath: absPath,
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

	fileInfo, err := os.Stat(absPath)
	if err == nil {
		// Can open in write mode?
		f, err := os.OpenFile(absPath, os.O_WRONLY, 0666)
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

	file, err := os.Open(absPath)
	if err != nil {
		b.Rows = append(b.Rows, []rune{})
		return b, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var rawLines []string
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

func (b *Buffer) Save() error {
	if b.Scratch || b.ReadOnly || b.AbsPath == "" {
		return errors.New("cannot save scratch, readonly, or unnamed buffer")
	}

	// TODO: Ask nicely before creating new file
	file, err := os.Create(b.AbsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	lineEnd := "\n"
	if b.Format == FormatDOS {
		lineEnd = "\r\n"
	}

	for i, row := range b.Rows {
		_, err := writer.WriteString(string(row))
		if err != nil {
			return err
		}

		// TODO: if last is empty add newline
		if i < len(b.Rows)-1 || len(row) > 0 {
			_, err = writer.WriteString(lineEnd)
			if err != nil {
				return err
			}
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	// TODO: Make actual hash system to be sure they are different
	b.Dirty = false
	return nil
}
