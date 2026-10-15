package spatial

import (
	"math"
	"testing"

	"github.com/gravitton/assert"
)

func TestNewHash(t *testing.T) {
	t.Run("keeps the cell size", func(t *testing.T) {
		assert.Equal(t, NewHash[int](32).CellSize(), 32)
		assert.Equal(t, NewHash[int](0.5).CellSize(), 0.5)
	})
	t.Run("panics for a cell size that is not positive and finite", func(t *testing.T) {
		assert.Panics(t, func() {
			NewHash[int](0)
		})
		assert.Panics(t, func() {
			NewHash[int](-1)
		})
		assert.Panics(t, func() {
			NewHash[int](math.NaN())
		})
		assert.Panics(t, func() {
			NewHash[int](math.Inf(1))
		})
	})
}

func TestHash_CellSize(t *testing.T) {
	assert.Equal(t, NewHash[string, float32](16).CellSize(), float32(16))
}
