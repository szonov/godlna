package fs_util

import (
	"errors"
	"io"
	"log/slog"
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

func CopyFile(src, dst string) (err error) {
	var source *os.File
	source, err = os.Open(src)
	if err != nil {
		return
	}
	defer func(source *os.File) {
		err = source.Close()
		if err != nil {
			slog.Error("util.CopyFile close source", "err", err.Error())
		}
	}(source)

	err = EnsureDirectoryExistsForFile(dst)
	if err != nil {
		return
	}

	destination, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func(destination *os.File) {
		err = destination.Close()
		if err != nil {
			slog.Error("util.CopyFile close destination", "err", err.Error())
		}
	}(destination)

	_, err = io.Copy(destination, source)
	if err != nil {
		return
	}
	err = destination.Sync()
	return
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
