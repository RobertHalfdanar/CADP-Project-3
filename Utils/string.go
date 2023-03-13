package Utils

import "unicode"

// IsValidCommand returns true if the string is a valid command
// A valid command is a string that does not contain any punctuation except dashes and underscores
func IsValidCommand(s string) bool {
	for _, r := range s {
		// Spaces are not allowed
		if r == ' ' {
			return false
		}

		// Allow dashes and underscores
		if r == '-' || r == '_' {
			continue
		}

		if unicode.IsPunct(r) {
			return false
		}
	}
	return true
}

