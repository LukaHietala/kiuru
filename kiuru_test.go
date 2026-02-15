package main

import (
	"crypto/rand"
	"os"
	"testing"
)

func TestAddBuffer(t *testing.T) {
	e := &Editor{Buffers: []*Buffer{}}

	// Test adding a listed buffer
	e.addBuffer(true, false)
	if len(e.Buffers) != 1 {
		t.Errorf("Expected 1 buffer, got %d", len(e.Buffers))
	}
	if e.curBuf().Name != "No name" {
		t.Errorf("Expected name 'No name', got %s", e.curBuf().Name)
	}

	// Test adding a scratch buffer
	e.addBuffer(false, true)
	if len(e.Buffers) != 2 {
		t.Errorf("Expected 2 buffers, got %d", len(e.Buffers))
	}
	if e.curBuf().Name != "Scratch" {
		t.Errorf("Expected name 'Scratch', got %s", e.curBuf().Name)
	}

	// Check if BufIndex updated correctly
	if e.BufIndex != 1 {
		t.Errorf("Expected BufIndex 1, got %d", e.BufIndex)
	}
}

func TestCurrentBuffer(t *testing.T) {
	e := &Editor{Buffers: []*Buffer{}}

	if e.curBuf() != nil {
		t.Error("curBuf() should return nil when no buffers exist")
	}

	e.addBuffer(true, false)
	if e.curBuf() == nil {
		t.Error("curBuf() should not be nil after adding a buffer")
	}
}

func TestOpenFile(t *testing.T) {
	e := &Editor{Buffers: []*Buffer{}}

	// Test (hopefully) non-existent file
	e.openFile(rand.Text())
	if len(e.curBuf().Rows) != 1 {
		t.Errorf("Expected 1 row on non-existent file, got %d", len(e.curBuf().Rows))
	}

	// Create file to test on
	f, err := os.CreateTemp("", "kiuru-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Check if buf name same as path
	e.openFile(f.Name())
	if e.curBuf().Name != f.Name() {
		t.Errorf("Expected buffer name to be %s, got %s", f.Name(), e.curBuf().Name)
	}

	// Ensure there is only one row
	if len(e.curBuf().Rows) != 1 {
		t.Errorf("Expected row lenght to be 1, got %d", len(e.curBuf().Rows))
	}

	// Test with file content
	_, err = f.WriteString("yskiva\npomeranian")
	if err != nil {
		panic(err)
	}
	e.openFile(f.Name())

	// Ensure rows match with file content
	if string(e.curBuf().Rows[0]) != "yskiva" {
		t.Errorf("Expected row 0 to be 'yskiva', got %s", string(e.curBuf().Rows[0]))
	}
	if string(e.curBuf().Rows[1]) != "pomeranian" {
		t.Errorf("Expected row 1 to be 'pomeranian', got %s", string(e.curBuf().Rows[1]))
	}

	// Test that the buf index and count is correct
	if e.BufIndex != 2 {
		t.Errorf("Expected BufIndex 2, got %d", e.BufIndex)
	}
	if len(e.Buffers) != 3 {
		t.Errorf("Expected 3 buffers, got %d", len(e.Buffers))
	}

}
