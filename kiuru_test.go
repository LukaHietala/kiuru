package main

import (
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
