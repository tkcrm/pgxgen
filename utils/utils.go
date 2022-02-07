package utils

func ExistInStringArray(values []string, item string) bool {
	for _, value := range values {
		if value == item {
			return true
		}
	}
	return false
}
