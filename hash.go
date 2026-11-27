package spatial

import (
	"fmt"
	"iter"
	"math"
	"slices"

	geom "github.com/gravitton/geometry"
)

// bucket is the slots of the keys in one cell, kept in a slice so that walking every cell follows
// the changes made to the hash rather than the order of a map.
type bucket struct {
	cell  cell
	slots slotList
}

// Hash is a uniform spatial hash: keys stored with their boxes in square cells of one size over an
// unbounded plane.
//
// The zero value is not usable; NewHash builds one.
type Hash[K comparable, T geom.Number] struct {
	cellSize T
	slots    []slot[K, T]
	index    map[K]int
	cells    map[cell]int
	buckets  []bucket
	large    slotList
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
		cells:    map[cell]int{},
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
	h.buckets = h.buckets[:0]
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
			b := h.fill(cell{x, y})
			b.slots = append(b.slots, i)
		}
	}
}

// fill returns the bucket of the cell, taking the next one for a cell that held no key, with the
// list of a cell emptied before where one is kept past the end.
func (h *Hash[K, T]) fill(c cell) *bucket {
	if n, ok := h.cells[c]; ok {
		return &h.buckets[n]
	}

	n := len(h.buckets)
	h.cells[c] = n
	h.buckets = slices.Grow(h.buckets, 1)[:n+1]

	b := &h.buckets[n]
	b.cell, b.slots = c, b.slots[:0]

	return b
}

// unplace removes the slot's key from the cells it is over.
func (h *Hash[K, T]) unplace(i int) {
	s := &h.slots[i]
	if s.large {
		h.large = h.large.without(i)

		return
	}

	for y := s.cells.min.y; y <= s.cells.max.y; y++ {
		for x := s.cells.min.x; x <= s.cells.max.x; x++ {
			n := h.cells[cell{x, y}]
			b := &h.buckets[n]
			b.slots = b.slots.without(i)
			if len(b.slots) == 0 {
				h.drop(n)
			}
		}
	}
}

// drop removes the emptied bucket and its cell, the last bucket moved into its place. The emptied
// one is kept past the end, so the next cell to fill takes its list and a key moving across a
// border allocates nothing.
func (h *Hash[K, T]) drop(n int) {
	delete(h.cells, h.buckets[n].cell)

	last := len(h.buckets) - 1
	if n != last {
		h.buckets[n], h.buckets[last] = h.buckets[last], h.buckets[n]
		h.cells[h.buckets[n].cell] = n
	}

	h.buckets = h.buckets[:last]
}

// renumber points the lists the slot's key is in from one slot to another.
func (h *Hash[K, T]) renumber(from, to int) {
	s := &h.slots[to]
	if s.large {
		h.large.replace(from, to)

		return
	}

	for y := s.cells.min.y; y <= s.cells.max.y; y++ {
		for x := s.cells.min.x; x <= s.cells.max.x; x++ {
			h.buckets[h.cells[cell{x, y}]].slots.replace(from, to)
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
			cell{columnExact(int64(box.Min.X), size), columnExact(int64(box.Min.Y), size)},
			cell{columnExact(int64(box.Max.X), size), columnExact(int64(box.Max.Y), size)},
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

	if cells.area() <= int64(len(h.buckets)) {
		for y := cells.min.y; y <= cells.max.y; y++ {
			for x := cells.min.x; x <= cells.max.x; x++ {
				c := cell{x, y}
				h.eachTouchingIn(c, h.slotsIn(c), cells, box, visit)
			}
		}

		return
	}

	for _, b := range h.buckets {
		if cells.contains(b.cell) {
			h.eachTouchingIn(b.cell, b.slots, cells, box, visit)
		}
	}
}

// eachTouchingIn calls visit with the keys of the cell's list that eachTouching reports there.
func (h *Hash[K, T]) eachTouchingIn(c cell, list slotList, cells cellRange, box geom.Box[T], visit func(i int)) {
	for _, i := range list {
		if s := &h.slots[i]; s.firstIn(cells) == c && s.box.IntersectsBox(box) {
			visit(i)
		}
	}
}

// slotsIn returns the slots of the keys in the cell, none for a cell that holds no key.
func (h *Hash[K, T]) slotsIn(c cell) slotList {
	if n, ok := h.cells[c]; ok {
		return h.buckets[n].slots
	}

	return nil
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
			for _, i := range h.slotsIn(at) {
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
	for _, filled := range h.buckets {
		for n, i := range filled.slots {
			a := &h.slots[i]
			for _, j := range filled.slots[n+1:] {
				b := &h.slots[j]
				if a.firstIn(b.cells) == filled.cell && a.box.IntersectsBox(b.box) {
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
