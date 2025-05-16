package utils

// Max returns the larger of x or y
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// Min returns the smaller of x or y
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
