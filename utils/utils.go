package utils

import "os"

func IsExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func IsNotExist(filename string) bool {
	return !IsExist(filename)
}
