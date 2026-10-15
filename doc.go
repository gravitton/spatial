// Package spatial holds broad-phase spatial indexes for games. An index stores a key with a
// geom.Box and answers which keys may touch a region; the caller narrows the candidates with the
// shapes of github.com/gravitton/geometry.
//
// # Types
//
// [Hash] is a uniform spatial hash: an unbounded plane of square cells, for keys of about the
// same size.
//
// # Panics
//
// A degenerate value gets an answer; an argument with no meaning gets a panic:
//
//   - [NewHash] for a cell size that is not positive and finite: no cell can be drawn at it.
package spatial
