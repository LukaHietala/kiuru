package buffer

// TODO:
// - Commit (single change), redo, undo
// - Grouping, maybe Begin and End than every commit int between gets compresses
// to one node
// - String() for tree and maybe interactive one someday
// - Maybe even sqlite for presistence and other components of the app :D

type Change struct {
	firstLine int      // Index of the first line in change
	oldLines  [][]byte // Lines replaces/edited
	newLines  [][]byte // Lines added/modified
	cursorX   int      // Cursor X before
	cursorY   int      // Cursor Y before
}

type UndoNode struct {
	id       int
	parent   *UndoNode
	children []*UndoNode
	changes  []Change // Changes in single step
	cursorX  int      // Cursor X after
	cursorY  int      // Cursor Y after
}

type UndoTree struct {
	root    *UndoNode
	current *UndoNode
	nextID  int
}
