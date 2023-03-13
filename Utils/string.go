package Utils

import "unicode"

// HasPunctuation returns true if the string contains any punctuation
func HasPunctuation(s string) bool {
	for _, r := range s {
		if unicode.IsPunct(r) {
			return true
		}
	}
	return false
}
