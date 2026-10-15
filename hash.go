package spatial

import (
	geom "github.com/gravitton/geometry"
)

// Hash is a uniform spatial hash: keys stored with their boxes in square cells of one size over an
// unbounded plane. The zero value is not usable; NewHash builds one.
type Hash[K comparable, T geom.Number] struct {
	cellSize T
}

// NewHash returns an empty hash with square cells of the given size. It panics for a cell size
// that is not positive and finite.
func NewHash[K comparable, T geom.Number](cellSize T) *Hash[K, T] {
	if !(cellSize > 0) || !geom.IsFinite(cellSize) {
		panic("spatial: cell size must be positive and finite")
	}

	return &Hash[K, T]{cellSize: cellSize}
}

// CellSize returns the side of a cell.
func (h *Hash[K, T]) CellSize() T {
	return h.cellSize
}
