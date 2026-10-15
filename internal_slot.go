package spatial

import (
	geom "github.com/gravitton/geometry"
)

// slot is one stored key: its box, the cells it is spread over, and whether it is large instead,
// kept in the one list every query scans.
type slot[K comparable, T geom.Number] struct {
	key   K
	box   geom.Box[T]
	cells cellRange
	large bool
}

// firstIn returns the first cell of the range the slot's cells and the given ones share, the one
// cell a key spread over several is reported in.
func (s *slot[K, T]) firstIn(cells cellRange) cell {
	return cell{max(s.cells.min.x, cells.min.x), max(s.cells.min.y, cells.min.y)}
}
