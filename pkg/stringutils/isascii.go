package stringutils

// IsASCII checks if given string is ascii characters only
func IsASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}
