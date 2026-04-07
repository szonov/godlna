package ssdp

import (
	"os"
	"syscall"
	"time"
)

func IsSocket(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	_, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	mode := info.Mode()
	return mode&os.ModeSocket != 0
}

func runPeriodic(callback func(), interval time.Duration, stop <-chan struct{}) {

	callback()

	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		callback()
	}
}
