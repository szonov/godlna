package fs_util

import (
	"errors"
	"io"
	"log/slog"
	"os"
)

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
