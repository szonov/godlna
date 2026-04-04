//go:build linux

package fswatcher

import (
	"fmt"
	"strings"
	"sync"
)

type watchMap struct {
	pathByWd map[uint32]string // wd → pathname
	wdByPath map[string]uint32 // pathname → wd
	mu       sync.RWMutex
}

func newWatchMap() *watchMap {
	return &watchMap{
		pathByWd: make(map[uint32]string),
		wdByPath: make(map[string]uint32),
	}
}

func (w *watchMap) add(wd uint32, path string) {
	w.mu.Lock()
	w.pathByWd[wd] = path
	w.wdByPath[path] = wd
	w.mu.Unlock()
}

func (w *watchMap) path(wd uint32) (string, bool) {
	w.mu.RLock()
	path, ok := w.pathByWd[wd]
	w.mu.RUnlock()
	return path, ok
}

func (w *watchMap) rename(oldPath string, newPath string) {
	w.mu.Lock()
	for wd, path := range w.pathByWd {
		var to string
		if path == oldPath {
			to = newPath
		} else if strings.HasPrefix(path, oldPath+"/") {
			to = newPath + path[len(oldPath):]
		}
		if to != "" {
			delete(w.wdByPath, path)
			w.pathByWd[wd] = to
			w.wdByPath[to] = wd
		}
	}
	w.mu.Unlock()
}

func (w *watchMap) rmByPath(path string) {
	w.mu.Lock()
	if wd, ok := w.wdByPath[path]; ok {
		delete(w.pathByWd, wd)
		delete(w.wdByPath, path)
	}
	w.mu.Unlock()
}

func (w *watchMap) rmByPathRecursive(path string) []uint32 {
	wds := make([]uint32, 0)

	w.mu.Lock()
	for wd, p := range w.pathByWd {
		if p == path || strings.HasPrefix(p, path+"/") {
			delete(w.pathByWd, wd)
			delete(w.wdByPath, p)
			wds = append(wds, wd)
		}
	}
	w.mu.Unlock()

	return wds
}

func (w *watchMap) reset() {
	w.mu.Lock()
	w.pathByWd = make(map[uint32]string)
	w.wdByPath = make(map[string]uint32)
	w.mu.Unlock()
}

func (w *watchMap) debug() {
	w.mu.RLock()
	fmt.Printf("[pathByWd]\n")
	for wd, path := range w.pathByWd {
		fmt.Printf(" -> %d: %s\n", wd, path)
	}

	fmt.Printf("[wdByPath]\n")
	for path, wd := range w.wdByPath {
		fmt.Printf(" -> %s: %d\n", path, wd)
	}
	w.mu.RUnlock()
}
