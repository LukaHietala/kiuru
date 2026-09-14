package buffer

import "strings"

func (b *Buffer) Flags() BufferFlag {
	return b.flags
}

func (b *Buffer) HasFlag(f BufferFlag) bool {
	return b.flags&f != 0
}

func (b *Buffer) EnableFlag(f BufferFlag) {
	b.flags |= f
}

func (b *Buffer) DisableFlag(f BufferFlag) {
	b.flags &^= f
}

func (b *Buffer) ToggleFlag(f BufferFlag) {
	b.flags ^= f
}

func (f BufferFlag) String() string {
	if f == 0 {
		return ""
	}

	var parts []string
	if f&FlagNew != 0 {
		parts = append(parts, "new")
	}
	if f&FlagReadonly != 0 {
		parts = append(parts, "readonly")
	}
	// No scratch because it's already in the name :D

	return strings.Join(parts, "|")
}
