package spatial

import (
	"fmt"
	"iter"
	"math"
	"slices"

	geom "github.com/gravitton/geometry"
)

// Hash is a uniform spatial hash: keys stored with their boxes in square cells of one size over an
// unbounded plane.
//
// The zero value is not usable; NewHash builds one.
type Hash[K comparable, T geom.Number] struct {
	cellSize T
	slots    []slot[K, T]
	index    map[K]int
	cells    map[cell][]int
	spare    [][]int
	large    []int
	occupied cellRange
}

// NewHash returns an empty hash with square cells of the given size. It panics for a cell size
// that is not positive and finite.
func NewHash[K comparable, T geom.Number](cellSize T) *Hash[K, T] {
	if !(cellSize > 0) || !geom.IsFinite(cellSize) {
		panic(fmt.Errorf("%w %v: must be positive and finite", ErrInvalidCellSize, cellSize))
	}

	return &Hash[K, T]{
		cellSize: cellSize,
		index:    map[K]int{},
		cells:    map[cell][]int{},
		occupied: noCells,
	}
}

// CellSize returns the side of a cell.
func (h *Hash[K, T]) CellSize() T {
	return h.cellSize
}

// Set stores the key with the box, replacing the box it had.
func (h *Hash[K, T]) Set(key K, box geom.Box[T]) {
	i, ok := h.index[key]
	if !ok {
		h.index[key] = len(h.slots)
		h.slots = append(h.slots, slot[K, T]{key: key})
		h.place(len(h.slots)-1, box)

		return
	}

	s := &h.slots[i]
	if s.box == box {
		return
	}

	cells, large := h.cellsOf(box)
	if large == s.large && (large || cells == s.cells) {
		s.box = box

		return
	}

	h.unplace(i)
	h.place(i, box)
}

// Delete removes the key; a key not stored is ignored.
func (h *Hash[K, T]) Delete(key K) {
	i, ok := h.index[key]
	if !ok {
		return
	}

	h.unplace(i)

	last := len(h.slots) - 1
	if i != last {
		h.slots[i] = h.slots[last]
		h.index[h.slots[i].key] = i
		h.renumber(last, i)
	}

	h.slots[last] = slot[K, T]{}
	h.slots = h.slots[:last]
	delete(h.index, key)
}

// Clear removes every key.
func (h *Hash[K, T]) Clear() {
	clear(h.slots)
	h.slots = h.slots[:0]
	clear(h.index)
	clear(h.cells)
	h.spare = nil
	h.large = h.large[:0]
	h.occupied = noCells
}

// place stores the slot's key over the cells of the box.
func (h *Hash[K, T]) place(i int, box geom.Box[T]) {
	cells, large := h.cellsOf(box)

	s := &h.slots[i]
	s.box, s.cells, s.large = box, cells, large

	if large {
		h.large = append(h.large, i)

		return
	}

	h.occupied = h.occupied.union(cells)
	for y := cells.min.y; y <= cells.max.y; y++ {
		for x := cells.min.x; x <= cells.max.x; x++ {
			c := cell{x, y}
			list, ok := h.cells[c]
			if !ok && len(h.spare) > 0 {
				list = h.spare[len(h.spare)-1]
				h.spare = h.spare[:len(h.spare)-1]
			}
			h.cells[c] = append(list, i)
		}
	}
}

// unplace removes the slot's key from the cells it is over. A cell left empty is dropped and its
// list kept for the next cell to fill, so a key moving across a border allocates nothing.
func (h *Hash[K, T]) unplace(i int) {
	s := &h.slots[i]
	if s.large {
		h.large = without(h.large, i)

		return
	}

	for y := s.cells.min.y; y <= s.cells.max.y; y++ {
		for x := s.cells.min.x; x <= s.cells.max.x; x++ {
			c := cell{x, y}
			list := without(h.cells[c], i)
			if len(list) == 0 {
				delete(h.cells, c)
				h.spare = append(h.spare, list)
				continue
			}
			h.cells[c] = list
		}
	}
}

// renumber points the lists the slot's key is in from one slot to another.
func (h *Hash[K, T]) renumber(from, to int) {
	s := &h.slots[to]
	if s.large {
		replace(h.large, from, to)

		return
	}

	for y := s.cells.min.y; y <= s.cells.max.y; y++ {
		for x := s.cells.min.x; x <= s.cells.max.x; x++ {
			replace(h.cells[cell{x, y}], from, to)
		}
	}
}

// cellsOf returns the cells a key with the box is spread over, and whether it is large instead: too
// many cells, or a coordinate that is NaN, infinite or beyond the cells.
func (h *Hash[K, T]) cellsOf(box geom.Box[T]) (cellRange, bool) {
	if !isFinite(box) {
		return noCells, true
	}

	cells := h.rangeOf(box)
	if cells.isLarge() {
		return noCells, true
	}

	return cells, false
}

// rangeOf returns the cells within the tolerance of a finite box.
func (h *Hash[K, T]) rangeOf(box geom.Box[T]) cellRange {
	box = box.Canonical()

	if isInteger[T]() {
		size := int64(h.cellSize)

		return cellRange{
			cell{floorDiv(int64(box.Min.X), size), floorDiv(int64(box.Min.Y), size)},
			cell{floorDiv(int64(box.Max.X), size), floorDiv(int64(box.Max.Y), size)},
		}
	}

	a, b, size := box.Min.Float(), box.Max.Float(), float64(h.cellSize)
	margin := tolerance[T](magnitudeOf(a, b))

	return cellRange{
		cell{column(a.X-margin, size), column(a.Y-margin, size)},
		cell{column(b.X+margin, size), column(b.Y+margin, size)},
	}
}

// Get returns the box stored with the key, and false where the key is not stored.
func (h *Hash[K, T]) Get(key K) (geom.Box[T], bool) {
	i, ok := h.index[key]
	if !ok {
		return geom.Box[T]{}, false
	}

	return h.slots[i].box, true
}

// Len returns the number of keys stored.
func (h *Hash[K, T]) Len() int {
	return len(h.slots)
}

// All returns every key with its box, in no particular order. The hash must not change while it
// is ranged.
func (h *Hash[K, T]) All() iter.Seq2[K, geom.Box[T]] {
	return func(yield func(K, geom.Box[T]) bool) {
		for _, s := range h.slots {
			if !yield(s.key, s.box) {
				return
			}
		}
	}
}

// QueryBox returns the keys whose box touches the given one, in no particular order.
func (h *Hash[K, T]) QueryBox(box geom.Box[T]) []K {
	return h.AppendQueryBox(nil, box)
}

// AppendQueryBox appends the keys of QueryBox to dst and returns the extended slice.
func (h *Hash[K, T]) AppendQueryBox(dst []K, box geom.Box[T]) []K {
	if !isFinite(box) {
		for _, s := range h.slots {
			if s.box.IntersectsBox(box) {
				dst = append(dst, s.key)
			}
		}

		return dst
	}

	for _, i := range h.large {
		if s := &h.slots[i]; s.box.IntersectsBox(box) {
			dst = append(dst, s.key)
		}
	}

	h.eachTouching(box, func(i int) {
		dst = append(dst, h.slots[i].key)
	})

	return dst
}

// eachTouching calls visit with every key kept in cells whose box touches the finite box, once:
// in the first cell of the range the box's cells and its own share.
func (h *Hash[K, T]) eachTouching(box geom.Box[T], visit func(i int)) {
	cells, ok := h.rangeOf(box).intersect(h.occupied)
	if !ok {
		return
	}

	if cells.area() <= int64(len(h.cells)) {
		for y := cells.min.y; y <= cells.max.y; y++ {
			for x := cells.min.x; x <= cells.max.x; x++ {
				c := cell{x, y}
				h.eachTouchingIn(c, h.cells[c], cells, box, visit)
			}
		}

		return
	}

	for c, list := range h.cells {
		if cells.contains(c) {
			h.eachTouchingIn(c, list, cells, box, visit)
		}
	}
}

// eachTouchingIn calls visit with the keys of the cell's list that eachTouching reports there.
func (h *Hash[K, T]) eachTouchingIn(c cell, list []int, cells cellRange, box geom.Box[T], visit func(i int)) {
	for _, i := range list {
		if s := &h.slots[i]; s.firstIn(cells) == c && s.box.IntersectsBox(box) {
			visit(i)
		}
	}
}

// QueryPoint returns the keys whose box contains the point, in no particular order.
func (h *Hash[K, T]) QueryPoint(point geom.Point[T]) []K {
	return h.AppendQueryPoint(nil, point)
}

// AppendQueryPoint appends the keys of QueryPoint to dst and returns the extended slice.
func (h *Hash[K, T]) AppendQueryPoint(dst []K, point geom.Point[T]) []K {
	return h.AppendQueryBox(dst, geom.Box[T]{Min: point, Max: point})
}

// QuerySegment returns the keys whose box the segment meets, in no particular order. The hash must
// not change while it is ranged.
func (h *Hash[K, T]) QuerySegment(segment geom.Segment[T]) iter.Seq[K] {
	return func(yield func(K) bool) {
		start, end := segment.Start.Float(), segment.End.Float()
		h.walk(start, end.Subtract(start), 1, magnitudeOf(start, end), segment.IntersectsBox, yield)
	}
}

// AppendQuerySegment appends the keys of QuerySegment to dst and returns the extended slice.
func (h *Hash[K, T]) AppendQuerySegment(dst []K, segment geom.Segment[T]) []K {
	for key := range h.QuerySegment(segment) {
		dst = append(dst, key)
	}

	return dst
}

// QueryRay returns the keys whose box the ray meets, in no particular order. The hash must not
// change while it is ranged.
func (h *Hash[K, T]) QueryRay(ray geom.Ray[T]) iter.Seq[K] {
	return func(yield func(K) bool) {
		origin := ray.Origin.Float()
		h.walk(origin, ray.Vector.Float(), math.Inf(1), magnitudeOf(origin), ray.IntersectsBox, yield)
	}
}

// AppendQueryRay appends the keys of QueryRay to dst and returns the extended slice.
func (h *Hash[K, T]) AppendQueryRay(dst []K, ray geom.Ray[T]) []K {
	for key := range h.QueryRay(ray) {
		dst = append(dst, key)
	}

	return dst
}

// walk yields the keys whose box the cast meets, the large ones first and then the others at the
// first cell of theirs the cast passes, until yield returns false. A cast with a coordinate that is
// NaN or infinite has no cells to walk, and geom has it meet no box.
func (h *Hash[K, T]) walk(origin geom.Point[float64], direction geom.Vector[float64], reach, magnitude float64, meets func(geom.Box[T]) bool, yield func(K) bool) {
	for _, i := range h.large {
		if s := &h.slots[i]; meets(s.box) && !yield(s.key) {
			return
		}
	}

	if !geom.IsFinite(magnitude) || !geom.IsFinite(direction.X) || !geom.IsFinite(direction.Y) {
		return
	}

	c, ok := newCast[T](origin, direction, reach, magnitude, float64(h.cellSize), h.occupied)
	if !ok {
		return
	}

	previous := noSpan
	for y := c.first; ; y += c.step {
		current := c.row(y)

		more := c.cells(y, current, func(at cell) bool {
			for _, i := range h.cells[at] {
				s := &h.slots[i]
				if c.enters(s.cells, at, current, previous) && meets(s.box) && !yield(s.key) {
					return false
				}
			}

			return true
		})
		if !more || y == c.last {
			return
		}

		previous = current
	}
}

// Pairs returns every pair of keys whose boxes touch, each pair once in no particular order.
func (h *Hash[K, T]) Pairs() []Pair[K] {
	return h.AppendPairs(nil)
}

// AppendPairs appends the pairs of Pairs to dst and returns the extended slice.
func (h *Hash[K, T]) AppendPairs(dst []Pair[K]) []Pair[K] {
	for c, list := range h.cells {
		for n, i := range list {
			a := &h.slots[i]
			for _, j := range list[n+1:] {
				b := &h.slots[j]
				if a.firstIn(b.cells) == c && a.box.IntersectsBox(b.box) {
					dst = append(dst, Pair[K]{a.key, b.key})
				}
			}
		}
	}

	for n, i := range h.large {
		a := &h.slots[i]
		for _, j := range h.large[n+1:] {
			if b := &h.slots[j]; a.box.IntersectsBox(b.box) {
				dst = append(dst, Pair[K]{a.key, b.key})
			}
		}

		if !isFinite(a.box) {
			for j := range h.slots {
				if b := &h.slots[j]; !b.large && a.box.IntersectsBox(b.box) {
					dst = append(dst, Pair[K]{a.key, b.key})
				}
			}
			continue
		}

		h.eachTouching(a.box, func(j int) {
			dst = append(dst, Pair[K]{a.key, h.slots[j].key})
		})
	}

	return dst
}

// without returns the list with the value removed, the last value moved into its place.
func without(list []int, value int) []int {
	last := len(list) - 1
	list[slices.Index(list, value)] = list[last]

	return list[:last]
}

// replace replaces the value in the list with another.
func replace(list []int, from, to int) {
	list[slices.Index(list, from)] = to
}
