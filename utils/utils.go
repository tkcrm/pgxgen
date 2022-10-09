package utils

func ExistInArray[T comparable](values []T, item T) bool {
	for _, value := range values {
		if value == item {
			return true
		}
	}

	return false
}

func FilterString[T comparable](values []T, filterValues []T) []T {
	result := []T{}
	for _, item := range values {
		if ExistInArray(filterValues, item) {
			continue
		}
		result = append(result, item)
	}

	return result
}
