package fs_utils

import (
	"errors"
	"os"
	"path/filepath"
)

func FileExists(name string) bool {
	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func NameWithoutExtension(filename string) string {
	ext := filepath.Ext(filename)
	return filename[0 : len(filename)-len(ext)]
}
