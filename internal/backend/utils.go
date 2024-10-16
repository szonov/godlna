package backend

import (
	"path/filepath"
	"strings"
)

var videoExts = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

func isVideoFile(file string) bool {
	fileExt := strings.ToLower(filepath.Ext(file))
	for _, ext := range videoExts {
		if fileExt == ext {
			return true
		}
	}
	return false
}

func NameWithoutExt(file string) string {
	ext := filepath.Ext(file)
	return file[0 : len(file)-len(ext)]
}
