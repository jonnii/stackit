package utils

// ContainsString checks if a string is present in a slice of strings
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
