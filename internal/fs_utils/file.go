package fs_utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var VideoExtensions = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

func IsVideoFile(file string) bool {
	fileExt := strings.ToLower(filepath.Ext(file))
	for _, ext := range VideoExtensions {
		if fileExt == ext {
			return true
		}
	}
	return false
}

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
