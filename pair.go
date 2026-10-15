package spatial

// Pair is two keys whose boxes touch.
type Pair[K comparable] struct {
	A K
	B K
}
