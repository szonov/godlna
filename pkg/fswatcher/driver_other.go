//go:build !darwin && !linux

package fswatcher

import (
	"fmt"
)

type other struct{}

func newDriver() (driver, error) {
	return nil, fmt.Errorf("fswatcher not supported on the current platform")
}

func (w *other) addDirectory(dir string) error { return nil }
func (w *other) start() error                  { return nil }
func (w *other) stop() error                   { return nil }
func (w *other) withEventHandler(EventHandler) {}
func (w *other) withErrorHandler(ErrorHandler) {}
func (w *other) withIgnoreFn(IgnoreFn)         {}
func (w *other) watchList()                    { return nil }
