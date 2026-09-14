package buffer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func OpenFile(path string) (*Buffer, error) {
	if path == "" {
		return New(), nil
	}

	buf := New()

	absPath, isReadOnly, err := validatePath(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if abs, absErr := filepath.Abs(path); absErr == nil {
				path = abs
			}
			buf.path = path
			buf.name = filepath.Base(path)
			buf.EnableFlag(FlagNew)
			return buf, nil
		}
		return nil, err
	}

	buf.path = absPath
	buf.name = filepath.Base(absPath)
	if isReadOnly {
		buf.EnableFlag(FlagReadonly)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// TODO: keep track if dos
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	buf.lines = bytes.Split(content, []byte("\n"))
	buf.initialHash = buf.CurrentHash()

	return buf, nil
}

// Validates path and its contents
func validatePath(path string) (string, bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", false, err
	}

	if info.IsDir() {
		// TODO: Open file explorer
		return "", false, fmt.Errorf("'%s' is a directory", path)
	}

	// TODO: too limiting?
	// https://pkg.go.dev/io/fs#FileMode
	if !info.Mode().IsRegular() {
		return "", true, fmt.Errorf("'%s' is not a regular file", path)
	}

	// TODO: Ignores group perms and might not work for every windows case
	var readonly bool
	if info.Mode().Perm()&0o200 == 0 {
		// TODO: Warn
		readonly = true
	}

	return abs, readonly, nil
}
