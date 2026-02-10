package utils

import (
	"os"
	"path"
)

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ChangeExt(filename string, newExt string) string {
	ext := path.Ext(filename)
	return filename[0:len(filename)-len(ext)] + "." + newExt
}
