package util

// BinarySearchFunc returns the smallest index i in [low, high) at which f(i) is true,
// assuming that on the range [low, high), f(i) == true implies f(i+1) == true.
func BinarySearchFunc(low, high int, f func(int) bool) int {
	// For this function to work with negative indices as well,
	// the power of 2 step based variation of binary search is used
	i := low - 1
	step := 1
	for step < (high - low) {
		step <<= 1
	}
	step >>= 1

	// We find the last index for which f(i) is false
	for ; step > 0; step >>= 1 {
		if i+step < high && !f(i+step) {
			i += step
		}
	}

	// Then f(i+1) will be true
	return i + 1
}
