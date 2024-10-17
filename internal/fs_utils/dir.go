package fs_utils

import (
	"os"
	"path"
)

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
