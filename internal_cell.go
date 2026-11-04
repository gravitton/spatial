package spatial

import (
	"math"

	geom "github.com/gravitton/geometry"
)

// maxCell is the largest cell index on either axis. A coordinate beyond the cells it reaches is
// clamped into the outermost one, so an index never overflows.
const maxCell = 1 << 30

// largeCells is the most cells a key is spread over; a key over more is kept in one list every
// query scans.
const largeCells = 256

// cell is the index of a cell, its column and row.
type cell struct {
	x, y int
}

// cellRange is the cells from min to max inclusive; one with min past max on an axis is empty.
type cellRange struct {
	min, max cell
}

// noCells is the empty range, the identity of union.
var noCells = cellRange{cell{1, 1}, cell{0, 0}}

// isEmpty reports whether the range holds no cell.
func (r cellRange) isEmpty() bool {
	return r.min.x > r.max.x || r.min.y > r.max.y
}

// contains reports whether the cell is in the range.
func (r cellRange) contains(c cell) bool {
	return r.min.x <= c.x && c.x <= r.max.x && r.min.y <= c.y && c.y <= r.max.y
}

// area returns the number of cells in a range that is not empty.
func (r cellRange) area() int64 {
	return (int64(r.max.x) - int64(r.min.x) + 1) * (int64(r.max.y) - int64(r.min.y) + 1)
}

// isLarge reports whether a key over the range is kept out of the cells: it is spread over more
// than largeCells, or reaches a clamped outermost cell, where its true extent is unknown.
func (r cellRange) isLarge() bool {
	return r.area() > largeCells || r.min.x == -maxCell || r.min.y == -maxCell || r.max.x == maxCell || r.max.y == maxCell
}

// intersect returns the cells in both ranges, and false where there are none.
func (r cellRange) intersect(other cellRange) (cellRange, bool) {
	common := cellRange{
		cell{max(r.min.x, other.min.x), max(r.min.y, other.min.y)},
		cell{min(r.max.x, other.max.x), min(r.max.y, other.max.y)},
	}

	return common, !common.isEmpty()
}

// union returns the smallest range holding both.
func (r cellRange) union(other cellRange) cellRange {
	if r.isEmpty() {
		return other
	}

	return cellRange{
		cell{min(r.min.x, other.min.x), min(r.min.y, other.min.y)},
		cell{max(r.max.x, other.max.x), max(r.max.y, other.max.y)},
	}
}

// clampCell returns a floored cell coordinate as an index, clamped to the cells there are.
func clampCell(f float64) int {
	if f <= -maxCell {
		return -maxCell
	}
	if f >= maxCell {
		return maxCell
	}

	return int(f)
}

// column returns the index of the cell a float coordinate is in, along either axis.
func column(v, size float64) int {
	return clampCell(math.Floor(v / size))
}

// floorDiv returns the cell of an integer coordinate exactly, where a float64 division can round
// a coordinate on a border into the cell beside it.
func floorDiv(v, size int64) int {
	q := v / size
	if v%size < 0 {
		q--
	}

	return clampCell(float64(q))
}

// tolerance returns the margin a box is grown by before its cells are taken: four times the
// boundary tolerance geom compares at the magnitude, so a touch geom admits across a cell border
// is never lost to rounding. It is zero for an integer T, which geom compares exactly.
func tolerance[T geom.Number](magnitude float64) float64 {
	switch geom.Epsilon[T]() {
	case 0:
		return 0
	case geom.Delta32:
		return float64(4 * max(geom.Delta32, magnitude*0x1p-22))
	default:
		return float64(4 * max(geom.Delta, magnitude*0x1p-51))
	}
}

// isInteger reports whether T is an integer type.
func isInteger[T geom.Number]() bool {
	return geom.Epsilon[T]() == 0
}

// isFinite reports whether no coordinate of the box is NaN or infinite.
func isFinite[T geom.Number](box geom.Box[T]) bool {
	return geom.IsFinite(box.Min.X) && geom.IsFinite(box.Min.Y) && geom.IsFinite(box.Max.X) && geom.IsFinite(box.Max.Y)
}

// magnitudeOf returns the largest absolute coordinate of the points, the magnitude geom takes its
// tolerance from.
func magnitudeOf(points ...geom.Point[float64]) float64 {
	magnitude := 0.0
	for _, point := range points {
		magnitude = max(magnitude, math.Abs(point.X), math.Abs(point.Y))
	}

	return magnitude
}
