package spatial

import (
	geom "github.com/gravitton/geometry"
)

// span is the columns from lo to hi inclusive; one with lo past hi is empty.
type span struct {
	lo, hi int
}

// noSpan is the empty span.
var noSpan = span{1, 0}

// overlaps reports whether the span holds a column from lo to hi.
func (s span) overlaps(lo, hi int) bool {
	return max(s.lo, lo) <= min(s.hi, hi)
}

// cast is the walk of a segment or ray over the occupied cells: the points origin + t·direction for
// t from from to to, clipped to the occupied cells grown by the margin.
//
// The walk takes the rows in the order the cast crosses them, and in each row the columns the cast
// passes within the margin, in the order it passes them. Every row's columns lie on the far side of
// the previous row's in both bounds, so the rows where the walk crosses the cells of a key are
// consecutive; that lets enters find the first walked cell of a key from the current and previous
// row alone.
type cast struct {
	origin    geom.Point[float64]
	direction geom.Vector[float64]
	from, to  float64
	margin    float64
	size      float64
	columns   span
	first     int
	last      int
	step      int
}

// newCast returns the walk of the points origin + t·direction for t from zero to reach over the
// occupied cells, and false where the cast misses them. The magnitude is that of the cast as geom
// takes it, which the margin is grown to.
func newCast[T geom.Number](origin geom.Point[float64], direction geom.Vector[float64], reach, magnitude, size float64, occupied cellRange) (cast, bool) {
	if occupied.isEmpty() {
		return cast{}, false
	}

	low := geom.Point[float64]{X: float64(occupied.min.x) * size, Y: float64(occupied.min.y) * size}
	high := geom.Point[float64]{X: float64(occupied.max.x+1) * size, Y: float64(occupied.max.y+1) * size}

	magnitude = max(magnitude, magnitudeOf(low, high))

	c := cast{
		origin:    origin,
		direction: direction,
		to:        reach,
		margin:    max(tolerance[T](magnitude), tolerance[float64](magnitude)),
		size:      size,
		columns:   span{occupied.min.x, occupied.max.x},
	}

	if direction.X == 0 && direction.Y == 0 {
		c.to = 0
	}

	if !c.clip(origin.X, direction.X, low.X, high.X) || !c.clip(origin.Y, direction.Y, low.Y, high.Y) {
		return cast{}, false
	}

	ya, yb := origin.Y+c.from*direction.Y, origin.Y+c.to*direction.Y
	lo := geom.Clamp(column(min(ya, yb)-c.margin, size), occupied.min.y, occupied.max.y)
	hi := geom.Clamp(column(max(ya, yb)+c.margin, size), occupied.min.y, occupied.max.y)

	c.first, c.last, c.step = lo, hi, 1
	if direction.Y < 0 {
		c.first, c.last, c.step = hi, lo, -1
	}

	return c, true
}

// clip narrows the parameters to those whose coordinate on one axis lies from low to high, grown by
// the margin, and reports whether any are left.
func (c *cast) clip(origin, direction, low, high float64) bool {
	low, high = low-c.margin, high+c.margin

	if direction == 0 {
		return low <= origin && origin <= high
	}

	ta, tb := (low-origin)/direction, (high-origin)/direction
	c.from, c.to = max(c.from, min(ta, tb)), min(c.to, max(ta, tb))

	return c.from <= c.to
}

// row returns the columns the walk passes in the row: those within the margin of the cast where it
// lies within the margin of the row, clipped to the occupied columns. Every row of the walk meets
// that part of the cast, the first and last by the margin they were taken with.
func (c *cast) row(y int) span {
	from, to := c.from, c.to

	if c.direction.Y != 0 {
		ta := (float64(y)*c.size - c.margin - c.origin.Y) / c.direction.Y
		tb := (float64(y+1)*c.size + c.margin - c.origin.Y) / c.direction.Y
		from, to = max(from, min(ta, tb)), min(to, max(ta, tb))
	}

	xa, xb := c.origin.X+from*c.direction.X, c.origin.X+to*c.direction.X

	return span{max(column(min(xa, xb)-c.margin, c.size), c.columns.lo), min(column(max(xa, xb)+c.margin, c.size), c.columns.hi)}
}

// cells calls visit with every cell the walk passes in the row, in the order it passes them, until
// visit returns false, and reports whether it never did.
func (c *cast) cells(y int, columns span, visit func(cell) bool) bool {
	if c.direction.X < 0 {
		for x := columns.hi; x >= columns.lo; x-- {
			if !visit(cell{x, y}) {
				return false
			}
		}

		return true
	}

	for x := columns.lo; x <= columns.hi; x++ {
		if !visit(cell{x, y}) {
			return false
		}
	}

	return true
}

// enters reports whether the cell is the first the walk passes of the range: the walk passes none
// of it in the previous row, and the cell is the first of it in the current one.
func (c *cast) enters(r cellRange, at cell, current, previous span) bool {
	before := at.y - c.step
	if r.min.y <= before && before <= r.max.y && previous.overlaps(r.min.x, r.max.x) {
		return false
	}

	if c.direction.X < 0 {
		return at.x == min(current.hi, r.max.x)
	}

	return at.x == max(current.lo, r.min.x)
}
