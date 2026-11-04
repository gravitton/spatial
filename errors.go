package spatial

import (
	"errors"
)

var (
	// ErrInvalidCellSize is wrapped by the panic value of a cell size that is not positive and
	// finite.
	ErrInvalidCellSize = errors.New("spatial: invalid cell size")
)
