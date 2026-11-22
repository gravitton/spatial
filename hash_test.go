package spatial

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/gravitton/assert"
	geom "github.com/gravitton/geometry"
)

func TestNewHash(t *testing.T) {
	t.Run("keeps the cell size", func(t *testing.T) {
		assert.Equal(t, NewHash[int](32).CellSize(), 32)
		assert.Equal(t, NewHash[int](0.5).CellSize(), 0.5)
	})
	t.Run("is empty", func(t *testing.T) {
		assert.Equal(t, NewHash[int](32).Len(), 0)
	})
	t.Run("panics for a cell size that is not positive and finite", func(t *testing.T) {
		assert.PanicsWith(t, func() {
			NewHash[int](0)
		}, ErrInvalidCellSize)
		assert.PanicsWith(t, func() {
			NewHash[int](-1)
		}, ErrInvalidCellSize)
		assert.PanicsWith(t, func() {
			NewHash[int](math.NaN())
		}, ErrInvalidCellSize)
		assert.PanicsWith(t, func() {
			NewHash[int](math.Inf(1))
		}, ErrInvalidCellSize)
	})
}

func TestHash_CellSize(t *testing.T) {
	assert.Equal(t, NewHash[string, float32](16).CellSize(), float32(16))
}

func TestHash_Set(t *testing.T) {
	t.Run("stores the box", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))

		box, ok := h.Get(1)
		assert.True(t, ok)
		assert.Equal(t, box, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		assert.Equal(t, h.Len(), 1)
	})
	t.Run("replaces the box", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		h.Set(1, geom.Bx(geom.Pt(51, 51), geom.Pt(54, 54)))

		box, _ := h.Get(1)
		assert.Equal(t, box, geom.Bx(geom.Pt(51, 51), geom.Pt(54, 54)))
		assert.Equal(t, h.Len(), 1)
		assert.Empty(t, h.QueryPoint(geom.Pt(2, 2)))
		assert.Equal(t, h.QueryPoint(geom.Pt(52, 52)), []int{1})
	})
	t.Run("replaces the box within the same cells", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		h.Set(1, geom.Bx(geom.Pt(5, 5), geom.Pt(8, 8)))

		assert.Empty(t, h.QueryPoint(geom.Pt(2, 2)))
		assert.Equal(t, h.QueryPoint(geom.Pt(6, 6)), []int{1})
	})
	t.Run("replaces a large box", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))
		h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(2000, 2000)))
		h.Set(1, geom.Bx(geom.Pt(5, 5), geom.Pt(8, 8)))

		assert.Empty(t, h.QueryPoint(geom.Pt(1500, 1500)))
		assert.Equal(t, h.QueryPoint(geom.Pt(6, 6)), []int{1})
	})
	t.Run("stores a box with a NaN coordinate that no query finds", func(t *testing.T) {
		h := NewHash[int](10.0)
		h.Set(1, geom.Bx(geom.Pt(math.NaN(), 0), geom.Pt(1.0, 1)))

		_, ok := h.Get(1)
		assert.True(t, ok)
		assert.Empty(t, h.QueryBox(geom.Bx(geom.Pt(-100.0, -100), geom.Pt(100.0, 100))))
		assert.Empty(t, h.Pairs())
	})
	t.Run("stores an infinite box that every query finds", func(t *testing.T) {
		h := NewHash[int](10.0)
		h.Set(1, geom.Bx(geom.Pt(math.Inf(-1), math.Inf(-1)), geom.Pt(math.Inf(1), math.Inf(1))))
		h.Set(2, geom.Bx(geom.Pt(1.0, 1), geom.Pt(2.0, 2)))

		assert.EqualUnordered(t, h.QueryPoint(geom.Pt(1.5, 1.5)), []int{1, 2})
		assert.Equal(t, h.Pairs(), []Pair[int]{{1, 2}})
	})
	t.Run("keeps the box already stored", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))

		assert.Equal(t, h.QueryPoint(geom.Pt(2, 2)), []int{1})
	})
	t.Run("stores a box beyond the cells", func(t *testing.T) {
		h := NewHash[int](1.0)
		h.Set(1, geom.Bx(geom.Pt(1e12, 1e12), geom.Pt(1e12+1, 1e12+1)))
		h.Set(2, geom.Bx(geom.Pt(-1e12, -1e12), geom.Pt(-1e12+1, -1e12+1)))

		assert.Equal(t, h.QueryPoint(geom.Pt(1e12, 1e12)), []int{1})
		assert.Equal(t, h.QueryPoint(geom.Pt(-1e12, -1e12)), []int{2})
	})
	t.Run("allocates nothing moving within cells already used", func(t *testing.T) {
		h := NewHash[int](10)
		a, b := geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)), geom.Bx(geom.Pt(11, 1), geom.Pt(14, 4))
		h.Set(1, a)
		h.Set(1, b)
		h.Set(1, a)

		assert.Allocates(t, func() {
			h.Set(1, b)
			h.Set(1, a)
		}, 0)
	})
}

func BenchmarkHash_Set(b *testing.B) {
	for _, layout := range benchmarkLayouts {
		b.Run(layout.name, func(b *testing.B) {
			h, boxes := benchmarkHash(layout)
			step := geom.Vec(20.0, 12)
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				key := i % len(boxes)
				if i/len(boxes)%2 == 0 {
					h.Set(key, boxes[key].Translate(step))
					continue
				}
				h.Set(key, boxes[key])
			}
		})
	}
}

func TestHash_Delete(t *testing.T) {
	t.Run("removes the key", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		h.Delete(1)

		_, ok := h.Get(1)
		assert.False(t, ok)
		assert.Equal(t, h.Len(), 0)
		assert.Empty(t, h.QueryPoint(geom.Pt(2, 2)))
	})
	t.Run("ignores a key not stored", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		h.Delete(2)

		assert.Equal(t, h.Len(), 1)
	})
	t.Run("keeps the other keys", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		h.Set(2, geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))
		h.Set(3, geom.Bx(geom.Pt(3, 3), geom.Pt(14, 14)))
		h.Set(4, geom.Bx(geom.Pt(-1000, -1000), geom.Pt(1000, 1000)))
		h.Delete(1)
		h.Delete(2)

		assert.EqualUnordered(t, h.QueryPoint(geom.Pt(4, 4)), []int{3, 4})
		assert.EqualUnordered(t, h.QueryPoint(geom.Pt(500, 500)), []int{4})
		assert.Equal(t, h.Pairs(), []Pair[int]{{4, 3}})
	})
	t.Run("allocates nothing", func(t *testing.T) {
		h := NewHash[int](10)
		box := geom.Bx(geom.Pt(1, 1), geom.Pt(14, 14))
		h.Set(1, box)
		h.Delete(1)
		h.Set(1, box)

		assert.Allocates(t, func() {
			h.Delete(1)
			h.Set(1, box)
		}, 0)
	})
}

func TestHash_Clear(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
	h.Set(2, geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))
	h.Clear()

	assert.Equal(t, h.Len(), 0)
	assert.Empty(t, h.QueryPoint(geom.Pt(2, 2)))
	assert.Empty(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(0, 0), geom.Vec(1, 1)))))

	h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
	assert.Equal(t, h.QueryPoint(geom.Pt(2, 2)), []int{1})
}

func TestHash_Get(t *testing.T) {
	h := NewHash[string](10)
	h.Set("a", geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))

	t.Run("returns the stored box", func(t *testing.T) {
		box, ok := h.Get("a")
		assert.True(t, ok)
		assert.Equal(t, box, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
	})
	t.Run("returns false for a key not stored", func(t *testing.T) {
		box, ok := h.Get("b")
		assert.False(t, ok)
		assert.Equal(t, box, geom.Box[int]{})
	})
}

func TestHash_Len(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
	h.Set(2, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
	h.Set(1, geom.Bx(geom.Pt(5, 5), geom.Pt(6, 6)))

	assert.Equal(t, h.Len(), 2)
}

func TestHash_All(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
	h.Set(2, geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))

	t.Run("yields every key with its box", func(t *testing.T) {
		boxes := map[int]geom.Box[int]{}
		for key, box := range h.All() {
			boxes[key] = box
		}

		assert.Equal(t, len(boxes), 2)
		assert.Equal(t, boxes[1], geom.Bx(geom.Pt(1, 1), geom.Pt(4, 4)))
		assert.Equal(t, boxes[2], geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))
	})
	t.Run("stops early", func(t *testing.T) {
		count := 0
		for range h.All() {
			count++
			break
		}

		assert.Equal(t, count, 1)
	})
}

func TestHash_QueryBox(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(10, 10)))
	h.Set(2, geom.Bx(geom.Pt(-25, -25), geom.Pt(-15, -15)))
	h.Set(3, geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))

	t.Run("returns the keys whose box touches the given one", func(t *testing.T) {
		assert.EqualUnordered(t, h.QueryBox(geom.Bx(geom.Pt(5, 5), geom.Pt(6, 6))), []int{1, 3})
	})
	t.Run("returns a key touching at an edge", func(t *testing.T) {
		assert.EqualUnordered(t, h.QueryBox(geom.Bx(geom.Pt(10, -5), geom.Pt(12, 0))), []int{1, 3})
	})
	t.Run("returns a key at negative coordinates", func(t *testing.T) {
		assert.EqualUnordered(t, h.QueryBox(geom.Bx(geom.Pt(-30, -30), geom.Pt(-20, -20))), []int{2})
	})
	t.Run("returns a key once over many cells", func(t *testing.T) {
		assert.EqualUnordered(t, h.QueryBox(geom.Bx(geom.Pt(-100, -100), geom.Pt(100, 100))), []int{1, 2, 3})
	})
	t.Run("returns nothing where no box is", func(t *testing.T) {
		assert.Empty(t, h.QueryBox(geom.Bx(geom.Pt(-100, 100), geom.Pt(-50, 200))))
	})
	t.Run("returns nothing from an empty hash", func(t *testing.T) {
		assert.Empty(t, NewHash[int](10).QueryBox(geom.Bx(geom.Pt(-100, 100), geom.Pt(-50, 200))))
	})
	t.Run("returns a key touching within the tolerance", func(t *testing.T) {
		h := NewHash[int](1.0)
		h.Set(1, geom.Bx(geom.Pt(0.0, 0), geom.Pt(1-geom.Delta/2, 1)))

		assert.Equal(t, h.QueryBox(geom.Bx(geom.Pt(1.0, 0), geom.Pt(2.0, 1))), []int{1})
	})
	t.Run("agrees with a linear scan", func(t *testing.T) {
		forEachType(t, func(t *testing.T, f fixture) {
			for range 300 {
				f.checkBox(t, randomBox(f.rng, f.cellSize))
			}
		})
	})
}

func TestHash_concurrentQueries(t *testing.T) {
	h, boxes := benchmarkHash(benchmarkLayout{n: 300})
	expected := h.Pairs()

	var group sync.WaitGroup
	for n := range 8 {
		group.Go(func() {
			for i := n; i < len(boxes); i += 8 {
				var keys []int
				keys = h.AppendQueryBox(keys, boxes[i])
				keys = h.AppendQueryPoint(keys, boxes[i].Min)
				keys = h.AppendQuerySegment(keys, geom.Seg(boxes[i].Min, boxes[(i+1)%len(boxes)].Max))
				keys = h.AppendQueryRay(keys, geom.Ry(boxes[i].Min, geom.Vec(1.0, 1)))
				assert.NotEmpty(t, keys)
			}
			assertPairs(t, h.Pairs(), expected)
		})
	}
	group.Wait()
}

func TestHash_AppendQueryBox(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(10, 10)))
	h.Set(2, geom.Bx(geom.Pt(0, 0), geom.Pt(1000, 1000)))
	query := geom.Bx(geom.Pt(5, 5), geom.Pt(30, 30))

	t.Run("appends the keys of QueryBox", func(t *testing.T) {
		keys := h.AppendQueryBox([]int{7}, query)
		assert.Equal(t, keys[0], 7)
		assert.EqualUnordered(t, keys[1:], h.QueryBox(query))
	})
	t.Run("allocates nothing with room", func(t *testing.T) {
		dst := make([]int, 0, 8)
		assert.Allocates(t, func() {
			sinkKeys = h.AppendQueryBox(dst, query)
		}, 0)
	})
}

func BenchmarkHash_AppendQueryBox(b *testing.B) {
	for _, layout := range benchmarkLayouts {
		b.Run("hash/"+layout.name, func(b *testing.B) {
			h, boxes := benchmarkHash(layout)
			dst := make([]int, 0, 64)
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				sinkKeys = h.AppendQueryBox(dst, boxes[i%len(boxes)])
			}
		})
		b.Run("scan/"+layout.name, func(b *testing.B) {
			_, boxes := benchmarkHash(layout)
			dst := make([]int, 0, 64)
			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				query := boxes[i%len(boxes)]
				dst = dst[:0]
				for key, box := range boxes {
					if box.IntersectsBox(query) {
						dst = append(dst, key)
					}
				}
				sinkKeys = dst
			}
		})
	}
}

func TestHash_QueryPoint(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(10, 10)))
	h.Set(2, geom.Bx(geom.Pt(10, 10), geom.Pt(20, 20)))

	t.Run("returns the keys whose box contains the point", func(t *testing.T) {
		assert.EqualUnordered(t, h.QueryPoint(geom.Pt(5, 5)), []int{1})
		assert.EqualUnordered(t, h.QueryPoint(geom.Pt(10, 10)), []int{1, 2})
		assert.Empty(t, h.QueryPoint(geom.Pt(25, 5)))
	})
	t.Run("agrees with a linear scan", func(t *testing.T) {
		forEachType(t, func(t *testing.T, f fixture) {
			for range 300 {
				f.checkPoint(t, randomPoint(f.rng, f.cellSize))
			}
		})
	})
}

func TestHash_AppendQueryPoint(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(10, 10)))

	t.Run("appends the keys of QueryPoint", func(t *testing.T) {
		assert.Equal(t, h.AppendQueryPoint([]int{7}, geom.Pt(5, 5)), []int{7, 1})
	})
	t.Run("allocates nothing with room", func(t *testing.T) {
		dst := make([]int, 0, 8)
		assert.Allocates(t, func() {
			sinkKeys = h.AppendQueryPoint(dst, geom.Pt(5, 5))
		}, 0)
	})
}

func TestHash_QuerySegment(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(20, 0), geom.Pt(30, 10)))
	h.Set(2, geom.Bx(geom.Pt(50, 0), geom.Pt(60, 10)))
	h.Set(3, geom.Bx(geom.Pt(0, 40), geom.Pt(10, 50)))
	h.Set(4, geom.Bx(geom.Pt(-1000, 1000), geom.Pt(1000, 2000)))

	t.Run("returns the keys whose box the segment meets", func(t *testing.T) {
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(0, 5), geom.Pt(55, 5)))), []int{1, 2})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(55, 5), geom.Pt(40, 5)))), []int{2})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(5, 0), geom.Pt(5, 1500)))), []int{3, 4})
	})
	t.Run("returns a key touched at a corner", func(t *testing.T) {
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(40, -10), geom.Pt(20, 10)))), []int{1})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(40, 0), geom.Pt(30, 10)))), []int{1})
	})
	t.Run("returns a key for a segment of zero length", func(t *testing.T) {
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(25, 5), geom.Pt(25, 5)))), []int{1})
	})
	t.Run("returns the keys across rows and columns of no key", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(15, 15)))
		h.Set(2, geom.Bx(geom.Pt(100, 100), geom.Pt(115, 115)))
		h.Set(3, geom.Bx(geom.Pt(0, 100), geom.Pt(15, 115)))
		h.Set(4, geom.Bx(geom.Pt(100, 0), geom.Pt(115, 15)))

		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(5, 5), geom.Pt(105, 105)))), []int{1, 2})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(105, 105), geom.Pt(5, 5)))), []int{1, 2})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(105, 5), geom.Pt(5, 105)))), []int{3, 4})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(5, 5), geom.Pt(105, 5)))), []int{1, 4})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(5, 105), geom.Pt(5, 5)))), []int{1, 3})
	})
	t.Run("returns no key moved or deleted from the rows and columns it held", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(15, 15)))
		h.Set(2, geom.Bx(geom.Pt(100, 100), geom.Pt(115, 115)))
		h.Set(3, geom.Bx(geom.Pt(100, 0), geom.Pt(115, 15)))
		h.Set(2, geom.Bx(geom.Pt(200, 200), geom.Pt(215, 215)))
		h.Delete(3)

		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(5, 5), geom.Pt(205, 205)))), []int{1, 2})
		assert.EqualUnordered(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(105, 5), geom.Pt(105, 105)))), []int{})
	})
	t.Run("returns nothing where the segment misses every cell", func(t *testing.T) {
		assert.Empty(t, slices.Collect(h.QuerySegment(geom.Seg(geom.Pt(-50, -50), geom.Pt(-40, 100)))))
	})
	t.Run("stops early", func(t *testing.T) {
		for _, segment := range []geom.Segment[int]{
			geom.Seg(geom.Pt(0, 5), geom.Pt(55, 5)),
			geom.Seg(geom.Pt(55, 5), geom.Pt(0, 5)),
			geom.Seg(geom.Pt(5, 0), geom.Pt(5, 1500)),
		} {
			count := 0
			for range h.QuerySegment(segment) {
				count++
				break
			}
			assert.Equal(t, count, 1)
		}
	})
	t.Run("returns a large key first", func(t *testing.T) {
		for key := range h.QuerySegment(geom.Seg(geom.Pt(5, 0), geom.Pt(5, 1500))) {
			assert.Equal(t, key, 4)
			break
		}
	})
	t.Run("agrees with a linear scan", func(t *testing.T) {
		forEachType(t, func(t *testing.T, f fixture) {
			for range 300 {
				f.checkSegment(t, randomSegment(f.rng, f.cellSize))
			}
		})
	})
}

func TestHash_AppendQuerySegment(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(20, 0), geom.Pt(30, 10)))
	h.Set(2, geom.Bx(geom.Pt(50, 0), geom.Pt(60, 10)))
	segment := geom.Seg(geom.Pt(0, 5), geom.Pt(55, 5))

	t.Run("appends the keys of QuerySegment", func(t *testing.T) {
		keys := h.AppendQuerySegment([]int{7}, segment)
		assert.Equal(t, keys[0], 7)
		assert.EqualUnordered(t, keys[1:], slices.Collect(h.QuerySegment(segment)))
	})
	t.Run("allocates nothing with room", func(t *testing.T) {
		dst := make([]int, 0, 8)
		assert.Allocates(t, func() {
			sinkKeys = h.AppendQuerySegment(dst, segment)
		}, 0)
	})
}

func BenchmarkHash_AppendQuerySegment(b *testing.B) {
	for _, layout := range benchmarkLayouts {
		for _, length := range benchmarkSegments {
			b.Run("hash/"+length.name+"/"+layout.name, func(b *testing.B) {
				h, boxes := benchmarkHash(layout)
				dst := make([]int, 0, 64)
				b.ResetTimer()
				for i := 0; b.Loop(); i++ {
					sinkKeys = h.AppendQuerySegment(dst, length.segment(boxes, i))
				}
			})
			b.Run("scan/"+length.name+"/"+layout.name, func(b *testing.B) {
				_, boxes := benchmarkHash(layout)
				dst := make([]int, 0, 64)
				b.ResetTimer()
				for i := 0; b.Loop(); i++ {
					segment := length.segment(boxes, i)
					dst = dst[:0]
					for key, box := range boxes {
						if segment.IntersectsBox(box) {
							dst = append(dst, key)
						}
					}
					sinkKeys = dst
				}
			})
		}
	}
}

func TestHash_QueryRay(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(20, 0), geom.Pt(30, 10)))
	h.Set(2, geom.Bx(geom.Pt(50, 0), geom.Pt(60, 10)))
	h.Set(3, geom.Bx(geom.Pt(0, 40), geom.Pt(10, 50)))

	t.Run("returns the keys whose box the ray meets", func(t *testing.T) {
		assert.EqualUnordered(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(0, 5), geom.Vec(1, 0)))), []int{1, 2})
		assert.EqualUnordered(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(40, 5), geom.Vec(1, 0)))), []int{2})
		assert.EqualUnordered(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(40, 5), geom.Vec(-1, 0)))), []int{1})
		assert.EqualUnordered(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(-5, -5), geom.Vec(1, 1)))), []int{})
		assert.EqualUnordered(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(-40, -40), geom.Vec(1, 2)))), []int{3})
	})
	t.Run("returns the keys at the origin of a ray of zero vector", func(t *testing.T) {
		assert.EqualUnordered(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(25, 5), geom.Vec(0, 0)))), []int{1})
		assert.Empty(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(-25, 5), geom.Vec(0, 0)))))
	})
	t.Run("returns nothing for a ray away from every cell", func(t *testing.T) {
		assert.Empty(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(0, 5), geom.Vec(-1, 0)))))
		assert.Empty(t, slices.Collect(h.QueryRay(geom.Ry(geom.Pt(-20, -50), geom.Vec(0, 1)))))
	})
	t.Run("agrees with a linear scan", func(t *testing.T) {
		forEachType(t, func(t *testing.T, f fixture) {
			for range 300 {
				f.checkRay(t, randomRay(f.rng, f.cellSize))
			}
		})
	})
}

func TestHash_AppendQueryRay(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(20, 0), geom.Pt(30, 10)))
	h.Set(2, geom.Bx(geom.Pt(50, 0), geom.Pt(60, 10)))
	ray := geom.Ry(geom.Pt(0, 5), geom.Vec(1, 0))

	t.Run("appends the keys of QueryRay", func(t *testing.T) {
		keys := h.AppendQueryRay([]int{7}, ray)
		assert.Equal(t, keys[0], 7)
		assert.EqualUnordered(t, keys[1:], slices.Collect(h.QueryRay(ray)))
	})
	t.Run("allocates nothing with room", func(t *testing.T) {
		dst := make([]int, 0, 8)
		assert.Allocates(t, func() {
			sinkKeys = h.AppendQueryRay(dst, ray)
		}, 0)
	})
}

func TestHash_Pairs(t *testing.T) {
	t.Run("returns every pair of keys whose boxes touch", func(t *testing.T) {
		h := NewHash[int](10)
		h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(15, 15)))
		h.Set(2, geom.Bx(geom.Pt(15, 15), geom.Pt(25, 25)))
		h.Set(3, geom.Bx(geom.Pt(40, 40), geom.Pt(45, 45)))
		h.Set(4, geom.Bx(geom.Pt(-1000, -1000), geom.Pt(0, 0)))
		h.Set(5, geom.Bx(geom.Pt(-2000, -2000), geom.Pt(-500, -500)))

		assertPairs(t, h.Pairs(), []Pair[int]{{1, 2}, {1, 4}, {4, 5}})
	})
	t.Run("returns nothing from an empty hash", func(t *testing.T) {
		assert.Empty(t, NewHash[int](10).Pairs())
	})
	t.Run("agrees with a linear scan", func(t *testing.T) {
		forEachType(t, func(t *testing.T, f fixture) {
			f.checkPairs(t)
		})
	})
}

func TestHash_AppendPairs(t *testing.T) {
	h := NewHash[int](10)
	h.Set(1, geom.Bx(geom.Pt(0, 0), geom.Pt(15, 15)))
	h.Set(2, geom.Bx(geom.Pt(15, 15), geom.Pt(25, 25)))
	h.Set(3, geom.Bx(geom.Pt(-1000, -1000), geom.Pt(0, 0)))

	t.Run("appends the pairs of Pairs", func(t *testing.T) {
		pairs := h.AppendPairs([]Pair[int]{{7, 8}})
		assert.Equal(t, pairs[0], Pair[int]{7, 8})
		assertPairs(t, pairs[1:], h.Pairs())
	})
	t.Run("allocates nothing with room", func(t *testing.T) {
		dst := make([]Pair[int], 0, 8)
		assert.Allocates(t, func() {
			sinkPairs = h.AppendPairs(dst)
		}, 0)
	})
}

func BenchmarkHash_AppendPairs(b *testing.B) {
	for _, layout := range benchmarkLayouts {
		b.Run("hash/"+layout.name, func(b *testing.B) {
			h, _ := benchmarkHash(layout)
			var dst []Pair[int]
			b.ResetTimer()
			for b.Loop() {
				dst = h.AppendPairs(dst[:0])
			}
			sinkPairs = dst
		})
	}
}

func FuzzHash(f *testing.F) {
	f.Add(uint64(1), []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	f.Add(uint64(7), []byte{9, 9, 9, 1, 1, 1, 200, 3, 3})
	f.Fuzz(func(t *testing.T, seed uint64, ops []byte) {
		rng := rand.New(rand.NewPCG(seed, seed))
		h := NewHash[int](4.0)
		stored := map[int]geom.Box[float64]{}

		for _, op := range ops {
			key := int(op % 16)
			switch op % 5 {
			case 0, 1:
				box := randomBox(rng, 4.0)
				h.Set(key, box)
				stored[key] = box
			case 2:
				h.Delete(key)
				delete(stored, key)
			case 3:
				query := randomBox(rng, 4.0)
				assert.EqualUnordered(t, h.QueryBox(query), scanKeys(stored, func(box geom.Box[float64]) bool {
					return box.IntersectsBox(query)
				}))
			default:
				query := randomSegment(rng, 4.0)
				assert.EqualUnordered(t, slices.Collect(h.QuerySegment(query)), scanKeys(stored, func(box geom.Box[float64]) bool {
					return query.IntersectsBox(box)
				}))
			}
		}

		assert.Equal(t, h.Len(), len(stored))
	})
}

func ExampleHash() {
	type entity int

	index := NewHash[entity](64.0)
	index.Set(1, geom.Bx(geom.Pt(10.0, 10), geom.Pt(40.0, 40)))
	index.Set(2, geom.Bx(geom.Pt(30.0, 30), geom.Pt(90.0, 50)))
	index.Set(3, geom.Bx(geom.Pt(500.0, 500), geom.Pt(520.0, 520)))

	fmt.Println(index.QueryPoint(geom.Pt(35.0, 35)))

	for _, pair := range index.Pairs() {
		fmt.Println(min(pair.A, pair.B), max(pair.A, pair.B))
	}

	for key := range index.QuerySegment(geom.Seg(geom.Pt(400.0, 600), geom.Pt(600.0, 400))) {
		fmt.Println(key)
	}

	// Output:
	// [1 2]
	// 1 2
	// 3
}

func ExampleHash_AppendQueryBox() {
	index := NewHash[int](64.0)
	index.Set(1, geom.Bx(geom.Pt(10.0, 10), geom.Pt(40.0, 40)))
	index.Set(2, geom.Bx(geom.Pt(100.0, 20), geom.Pt(120.0, 60)))
	index.Set(3, geom.Bx(geom.Pt(500.0, 500), geom.Pt(520.0, 520)))

	camera := geom.Bx(geom.Pt(0.0, 0), geom.Pt(200.0, 150))
	cursor := geom.Pt(110.0, 30)

	var keys []int

	keys = index.AppendQueryBox(keys[:0], camera)
	slices.Sort(keys)
	fmt.Println(keys)

	keys = index.AppendQueryPoint(keys[:0], cursor)
	fmt.Println(keys)

	// Output:
	// [1 2]
	// [2]
}

func ExampleHash_QuerySegment() {
	walls := map[int]geom.Shape[float64]{
		1: geom.Bx(geom.Pt(40.0, 0), geom.Pt(50.0, 30)),
		2: geom.Circ(geom.Pt(60.0, 60), 5),
	}

	index := NewHash[int](32.0)
	for key, wall := range walls {
		index.Set(key, wall.Bounds())
	}

	sight := geom.Seg(geom.Pt(0.0, 50), geom.Pt(100.0, 50))
	blocked := false
	for key := range index.QuerySegment(sight) {
		if geom.Intersects(walls[key], sight) {
			blocked = true
			break
		}
	}

	fmt.Println(blocked)

	// Output:
	// false
}

func ExampleHash_AppendPairs() {
	shapes := map[int]geom.Shape[float64]{
		1: geom.Circ(geom.Pt(10.0, 10), 5),
		2: geom.Circ(geom.Pt(17.0, 10), 5),
		3: geom.Bx(geom.Pt(21.0, 14), geom.Pt(30.0, 30)),
	}

	index := NewHash[int](16.0)
	for key, shape := range shapes {
		index.Set(key, shape.Bounds())
	}

	var pairs []Pair[int]
	pairs = index.AppendPairs(pairs[:0])
	for _, pair := range pairs {
		if _, ok := geom.Collide(shapes[pair.A], shapes[pair.B]); ok {
			fmt.Println(min(pair.A, pair.B), max(pair.A, pair.B))
		}
	}

	// Output:
	// 1 2
}

// assertPairs asserts that the pairs are the expected ones, each once in either order.
func assertPairs(t *testing.T, actual, expected []Pair[int]) {
	t.Helper()

	assert.EqualUnordered(t, orderedPairsOf(actual), orderedPairsOf(expected))
}

// forEachType runs the test on a fixture of int, float32 and float64.
func forEachType(t *testing.T, test func(t *testing.T, f fixture)) {
	t.Run("int", func(t *testing.T) {
		test(t, newFixture[int](8))
	})
	t.Run("float32", func(t *testing.T) {
		test(t, newFixture[float32](1.5))
	})
	t.Run("float64", func(t *testing.T) {
		test(t, newFixture[float64](1.5))
	})
}

// randomRay returns a ray from a random point along an axis, a diagonal, any direction, or of
// zero vector, sometimes infinite.
func randomRay(rng *rand.Rand, cellSize float64) geom.Ray[float64] {
	var vector geom.Vector[float64]
	switch n := rng.IntN(40); {
	case n == 0:
		vector = geom.Vec(math.Inf(1), 1)
	case n < 10:
		vector = geom.Vec(float64(rng.IntN(3)-1), 0)
	case n < 20:
		vector = geom.Vec(0, float64(rng.IntN(3)-1))
	case n < 30:
		vector = geom.Vec(float64(rng.IntN(2)*2-1), float64(rng.IntN(2)*2-1))
	default:
		vector = geom.Vec(float64(rng.IntN(21)-10), float64(rng.IntN(21)-10))
	}

	return geom.Ry(randomPoint(rng, cellSize), vector)
}

// randomPoint returns a random point near the origin, on a cell border, within the tolerance of
// one, or anywhere.
func randomPoint(rng *rand.Rand, cellSize float64) geom.Point[float64] {
	return geom.Pt(randomCoordinate(rng, cellSize), randomCoordinate(rng, cellSize))
}

// randomCoordinate returns a random coordinate within ten cells of the origin.
func randomCoordinate(rng *rand.Rand, cellSize float64) float64 {
	border := float64(rng.IntN(21)-10) * cellSize
	switch rng.IntN(4) {
	case 0:
		return border
	case 1:
		return border + []float64{-1, 1}[rng.IntN(2)]*geom.Delta32/2
	case 2:
		return border + []float64{-1, 1}[rng.IntN(2)]*geom.Delta/2
	default:
		return (rng.Float64()*20 - 10) * cellSize
	}
}

// fixture is a hash filled with random boxes by Set and Delete, with checks of each query against
// a linear scan of the boxes it stores, made in the type of the hash. The queries are given in
// float64 and cast into that type.
type fixture struct {
	rng          *rand.Rand
	cellSize     float64
	checkBox     func(t *testing.T, box geom.Box[float64])
	checkPoint   func(t *testing.T, point geom.Point[float64])
	checkSegment func(t *testing.T, segment geom.Segment[float64])
	checkRay     func(t *testing.T, ray geom.Ray[float64])
	checkPairs   func(t *testing.T)
}

// newFixture returns a fixture of a hash over T with the cell size.
func newFixture[T geom.Number](cellSize T) fixture {
	rng := rand.New(rand.NewPCG(1, 2))
	h := NewHash[int](cellSize)
	stored := map[int]geom.Box[T]{}

	for range 600 {
		key := rng.IntN(300)
		if rng.IntN(5) == 0 {
			h.Delete(key)
			delete(stored, key)
			continue
		}

		box := castBox[T](randomBox(rng, float64(cellSize)))
		h.Set(key, box)
		stored[key] = box
	}

	return fixture{
		rng:      rng,
		cellSize: float64(cellSize),
		checkBox: func(t *testing.T, box geom.Box[float64]) {
			query := castBox[T](box)
			assert.EqualUnordered(t, h.QueryBox(query), scanKeys(stored, func(box geom.Box[T]) bool {
				return box.IntersectsBox(query)
			}))
		},
		checkPoint: func(t *testing.T, point geom.Point[float64]) {
			query := castPoint[T](point)
			assert.EqualUnordered(t, h.QueryPoint(query), scanKeys(stored, func(box geom.Box[T]) bool {
				return box.Contains(query)
			}))
		},
		checkSegment: func(t *testing.T, segment geom.Segment[float64]) {
			query := geom.Segment[T]{Start: castPoint[T](segment.Start), End: castPoint[T](segment.End)}
			assert.EqualUnordered(t, slices.Collect(h.QuerySegment(query)), scanKeys(stored, func(box geom.Box[T]) bool {
				return query.IntersectsBox(box)
			}))
		},
		checkRay: func(t *testing.T, ray geom.Ray[float64]) {
			query := geom.Ray[T]{Origin: castPoint[T](ray.Origin), Vector: geom.Vector[T](castPoint[T](geom.Point[float64](ray.Vector)))}
			assert.EqualUnordered(t, slices.Collect(h.QueryRay(query)), scanKeys(stored, func(box geom.Box[T]) bool {
				return query.IntersectsBox(box)
			}))
		},
		checkPairs: func(t *testing.T) {
			var expected []Pair[int]
			for a, boxA := range stored {
				for b, boxB := range stored {
					if a < b && boxA.IntersectsBox(boxB) {
						expected = append(expected, Pair[int]{a, b})
					}
				}
			}

			assertPairs(t, h.Pairs(), expected)
		},
	}
}

// randomBox returns a box of random size at a random place, often on a cell border or within the
// tolerance of one, sometimes large, NaN or infinite.
func randomBox(rng *rand.Rand, cellSize float64) geom.Box[float64] {
	switch rng.IntN(40) {
	case 0:
		return geom.Bx(geom.Pt(math.NaN(), 0), randomPoint(rng, cellSize))
	case 1:
		return geom.Bx(geom.Pt(math.Inf(-1), -cellSize), randomPoint(rng, cellSize))
	case 2:
		return geom.Bx(randomPoint(rng, cellSize), randomPoint(rng, cellSize).Add(geom.Vec(30*cellSize, 30*cellSize)))
	}

	a := randomPoint(rng, cellSize)
	size := []float64{0, cellSize / 3, cellSize, 2.5 * cellSize}[rng.IntN(4)]

	return geom.Bx(a, a.Add(geom.Vec(size*rng.Float64(), size*rng.Float64())))
}

// castBox returns the box with each coordinate cast into T, rounded for an integer T, where an
// integer T keeps a coordinate that is no number at zero.
func castBox[T geom.Number](box geom.Box[float64]) geom.Box[T] {
	return geom.Box[T]{Min: castPoint[T](box.Min), Max: castPoint[T](box.Max)}
}

// castPoint returns the point cast into T as castBox does.
func castPoint[T geom.Number](point geom.Point[float64]) geom.Point[T] {
	return geom.Point[T]{X: castCoordinate[T](point.X), Y: castCoordinate[T](point.Y)}
}

// castCoordinate returns the coordinate cast into T as castBox does.
func castCoordinate[T geom.Number](v float64) T {
	if isInteger[T]() && !geom.IsFinite(v) {
		return 0
	}

	return geom.Cast[T](v)
}

// scanKeys returns the keys whose box passes the test.
func scanKeys[T geom.Number](stored map[int]geom.Box[T], test func(geom.Box[T]) bool) []int {
	keys := []int{}
	for key, box := range stored {
		if test(box) {
			keys = append(keys, key)
		}
	}

	return keys
}

// randomSegment returns a segment between random points, sometimes with a NaN or infinite end.
func randomSegment(rng *rand.Rand, cellSize float64) geom.Segment[float64] {
	end := randomPoint(rng, cellSize)
	switch rng.IntN(40) {
	case 0:
		end.X = math.NaN()
	case 1:
		end.Y = math.Inf(-1)
	}

	return geom.Seg(randomPoint(rng, cellSize), end)
}

// benchmarkLayout is a number of keys of about a cell, spread uniformly over a square world that
// keeps a few keys per cell, or in a few dense clusters over a world as wide or wider.
type benchmarkLayout struct {
	name      string
	n         int
	clustered bool
	scatter   float64
}

// benchmarkHash returns a hash of the layout and the boxes it stores.
func benchmarkHash(layout benchmarkLayout) (*Hash[int, float64], []geom.Box[float64]) {
	rng := rand.New(rand.NewPCG(3, 4))
	side := math.Sqrt(float64(layout.n)) * 32
	centers := make([]geom.Point[float64], 8)
	for i := range centers {
		centers[i] = geom.Pt(rng.Float64()*side*layout.scatter, rng.Float64()*side*layout.scatter)
	}

	h := NewHash[int](32.0)
	boxes := make([]geom.Box[float64], layout.n)
	for i := range boxes {
		a := geom.Pt(rng.Float64()*side, rng.Float64()*side)
		if layout.clustered {
			a = centers[i%len(centers)].Add(geom.Vec(rng.NormFloat64()*side/16, rng.NormFloat64()*side/16))
		}
		boxes[i] = geom.Bx(a, a.Add(geom.Vec(8+rng.Float64()*24, 8+rng.Float64()*24)))
		h.Set(i, boxes[i])
	}

	return h, boxes
}

// orderedPairsOf returns the pairs, the lesser key of each first.
func orderedPairsOf(pairs []Pair[int]) []Pair[int] {
	ordered := make([]Pair[int], len(pairs))
	for i, pair := range pairs {
		ordered[i] = Pair[int]{min(pair.A, pair.B), max(pair.A, pair.B)}
	}

	return ordered
}

// benchmarkSegment is a way to draw the i-th segment of a benchmark from the boxes stored.
type benchmarkSegment struct {
	name    string
	segment func(boxes []geom.Box[float64], i int) geom.Segment[float64]
}

var (
	sinkKeys  []int
	sinkPairs []Pair[int]

	// benchmarkSegments are the segments the cast benchmarks run with: a few cells from one of the
	// boxes, and from one box to the next, in another cluster where the layout has them.
	benchmarkSegments = []benchmarkSegment{
		{"short", func(boxes []geom.Box[float64], i int) geom.Segment[float64] {
			start := boxes[i%len(boxes)].Min

			return geom.Seg(start, start.Add(geom.Vec(150.0, 90)))
		}},
		{"long", func(boxes []geom.Box[float64], i int) geom.Segment[float64] {
			return geom.Seg(boxes[i%len(boxes)].Min, boxes[(i+1)%len(boxes)].Min)
		}},
	}

	// benchmarkLayouts are the layouts the benchmarks run at.
	benchmarkLayouts = []benchmarkLayout{
		{"uniform/30", 30, false, 1},
		{"uniform/300", 300, false, 1},
		{"uniform/30000", 30000, false, 1},
		{"clustered/300", 300, true, 1},
		{"clustered/30000", 30000, true, 1},
		{"sparse/300", 300, true, 16},
		{"sparse/30000", 30000, true, 16},
	}
)
