// Package spatial holds broad-phase spatial indexes for games. An index stores a key with a
// geom.Box and answers which keys may touch a region: the broad phase. It never stores a shape and
// never decides a collision; the caller narrows the candidates with the shapes of
// github.com/gravitton/geometry.
//
// # Types
//
// [Hash] is a uniform spatial hash: an unbounded plane of square cells, for keys of about the
// same size. Pick a cell about the size of a typical key; a key much larger than a cell costs every
// query a test, and a cell much larger than the keys makes a query test many keys it does not
// touch. [Pair] is two keys whose boxes touch.
//
// # Rules
//
// A key is a candidate exactly when geom says its box touches the region: Box.IntersectsBox for a
// box, Box.Contains for a point, Segment.IntersectsBox and Ray.IntersectsBox for a cast. The index
// never disagrees with the narrow phase on a touch, the tolerant one included.
//
// Every query reports a key at most once, and every pair once in either order. Results come in
// no particular order.
//
// A query never changes the index, so any number may run at once. A change must not run beside a
// query or another change, and a hash must not change while a query of it is ranged.
//
// A box with a NaN coordinate is kept by Get and All and found by no query, as geom has NaN touch
// nothing. A box reaching an infinity is found wherever geom says it touches.
//
// # Panics
//
// A degenerate value gets an answer; an argument with no meaning gets a panic:
//
//   - [NewHash] for a cell size that is not positive and finite: no cell can be drawn at it.
package spatial
