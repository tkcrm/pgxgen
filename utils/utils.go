package utils

func ExistInArray[V comparable](values []V, item V) bool {
	for _, value := range values {
		if value == item {
			return true
		}
	}
	return false
}
