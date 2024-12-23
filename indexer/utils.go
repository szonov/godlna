package indexer

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var videoExtensions = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

func IsPathIgnored(fullPath string) bool {
	checkPath := fullPath + "/"
	// dot files (hidden files) and synology special directories used for indexing
	if strings.Contains(checkPath, "/.") || strings.Contains(checkPath, "/@eaDir/") {
		return true
	}
	return false
}

func IsVideoFile(file string) bool {
	fileExt := strings.ToLower(filepath.Ext(file))
	for _, ext := range videoExtensions {
		if fileExt == ext {
			return true
		}
	}
	return false
}

func EnsureDirectoryExists(dir string) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		return err
	}
}

func EnsureDirectoryExistsForFile(file string) error {
	dir, _ := path.Split(file)
	return EnsureDirectoryExists(dir)
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

func DurationString(v int64) string {
	dur := time.Duration(v * int64(time.Millisecond))
	ms := dur.Milliseconds() % 1000
	s := int(dur.Seconds()) % 60
	m := int(dur.Minutes()) % 60
	h := int(dur.Hours())
	return fmt.Sprintf("%d:%02d:%02d.%03d", h, m, s, ms)
}
