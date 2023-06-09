package Utils

import (
	"net"
)

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

func Contains(e []*net.UDPAddr, item *net.UDPAddr) bool {
	for _, a := range e {
		if a.String() == item.String() {
			return true
		}
	}
	return false
}

func Find(s []*net.UDPAddr, e *net.UDPAddr) int {
	for i, a := range s {
		if a.String() == e.String() {
			return i
		}
	}
	return -1
}

func Remove(s []*net.UDPAddr, e *net.UDPAddr) []*net.UDPAddr {
	index := Find(s, e)

	if index == -1 {
		return s
	}

	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

func IndexOf(s []*net.UDPAddr, e *net.UDPAddr) int {
	for i, a := range s {
		if a.String() == e.String() {
			return i
		}
	}
	return -1
}

func Count() {

}
