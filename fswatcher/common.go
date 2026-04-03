package fswatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validatedAddDir(dir string, addedDirs []string) (string, error) {

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return absPath, fmt.Errorf("failed to get absolute path for '%s': %w", dir, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return absPath, fmt.Errorf("failed to stat path '%s': %w", absPath, err)
	}

	if !info.IsDir() {
		return absPath, fmt.Errorf("failed to add path '%s': non directory", absPath)
	}

	for _, v := range addedDirs {
		if absPath == v {
			return absPath, fmt.Errorf("failed to add path '%s': duplicate", absPath)
		}
		n := absPath + "/"
		o := v + "/"
		if strings.HasPrefix(n, o) {
			return absPath, fmt.Errorf("failed to add path '%s': is subdirectory of '%s'", absPath, v)
		}
		if strings.HasPrefix(o, n) {
			return absPath, fmt.Errorf("failed to add path '%s': has already added subdirectory '%s'", absPath, v)
		}
	}

	return absPath, nil
}
