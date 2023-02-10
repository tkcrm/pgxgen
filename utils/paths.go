package utils

import "os"

func CraetePath(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}
