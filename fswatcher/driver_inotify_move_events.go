//go:build linux

package fswatcher

import (
	"sync"
	"time"
)

type moveHandler func(op Op, oldName, newName string, isDir bool)

type mvFromEvent struct {
	name  string
	isDir bool
	done  chan struct{}
}

type mvEvents struct {
	mu      sync.Mutex
	mvFrom  map[uint32]*mvFromEvent
	handler moveHandler
	done    chan struct{}
}

func newMvEvents(handler moveHandler) *mvEvents {
	return &mvEvents{
		mvFrom:  map[uint32]*mvFromEvent{},
		handler: handler,
		done:    make(chan struct{}),
	}
}

func (e *mvEvents) addMvFrom(cookie uint32, name string, isDir bool) {
	done := make(chan struct{})

	e.mu.Lock()
	e.mvFrom[cookie] = &mvFromEvent{
		name:  name,
		isDir: isDir,
		done:  done,
	}
	e.mu.Unlock()

	go func() {
		select {
		case <-done:
		case <-e.done:
		case <-time.After(time.Millisecond * 100):
			e.handler(Remove, name, "", isDir)
		}
		e.mu.Lock()
		delete(e.mvFrom, cookie)
		e.mu.Unlock()
	}()

}

func (e *mvEvents) addMvTo(cookie uint32, name string, isDir bool) {
	var oldName string
	op := Index

	e.mu.Lock()
	mvFrom := e.mvFrom[cookie]
	if mvFrom != nil {
		oldName = mvFrom.name
		op = Rename
		close(mvFrom.done)
	}
	e.mu.Unlock()
	e.handler(op, oldName, name, isDir)
}

func (e *mvEvents) reset() {
	e.mvFrom = map[uint32]*mvFromEvent{}
	e.done = make(chan struct{})
	//fmt.Println("mv events reset")
}

func (e *mvEvents) stop() {
	close(e.done)
	//fmt.Println("mv events stopped")
}
