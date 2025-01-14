package tool

// IntToPointer
func IntToPointer(i int) *int {
	return &i
}

// IsStringInSlice function
func IsStringInSlice(s string, lists []string) bool {
	for _, v := range lists {
		if s == v {
			return true
		}
	}
	return false
}
