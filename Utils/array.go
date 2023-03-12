package Utils

// WrapIndex receives and index and a length and returns the index if it is in the range of the length
// otherwise wraps the index around the array
func WrapIndex(index int, length int) int {
	if index < 0 {
		return length + index
	} else if index >= length {
		return index - length
	}
	return index
}

func Contains(e []int32, item int32) bool {
	for _, a := range e {
		if a == item {
			return true
		}
	}
	return false
}
