package spatial

// Pair is two keys whose boxes touch, in no order to rely on.
type Pair[K comparable] struct {
	A K
	B K
}
