package fswatcher

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type delHandler func(name string, isDir bool)

type delItem struct {
	path  string
	isDir bool
}

type delEvent struct {
	items []delItem
	done  chan struct{}
}

type delEvents struct {
	mu      sync.Mutex
	ev      map[string]delEvent
	handler delHandler
	done    chan struct{}
}

func newDeleteEvents(handler delHandler) *delEvents {
	return &delEvents{
		ev:      map[string]delEvent{},
		handler: handler,
		done:    make(chan struct{}),
	}
}

func (e *delEvents) add(path string, isDir bool) {
	parentDir := filepath.Dir(path)
	done := make(chan struct{})
	items := make([]delItem, 0)

	e.mu.Lock()
	if evt, ok := e.ev[parentDir]; ok {
		close(evt.done)
		items = append(evt.items, delItem{path, isDir})
	} else {
		items = append(items, delItem{path, isDir})
	}
	if isDir {
		// remove directory children from eventing.
		// leave only events for root items
		for p, evt := range e.ev {
			if strings.HasPrefix(p, path+"/") || p == path {
				close(evt.done)
				delete(e.ev, p)
			}
		}
	}

	e.ev[parentDir] = delEvent{
		items: items,
		done:  done,
	}
	e.mu.Unlock()

	go func() {
		select {
		case <-done:
		case <-e.done:
		case <-time.After(time.Millisecond * 100):
			e.mu.Lock()
			delete(e.ev, parentDir)
			e.mu.Unlock()
			for _, item := range items {
				e.handler(item.path, item.isDir)
			}
		}
	}()
}

func (e *delEvents) reset() {
	e.ev = map[string]delEvent{}
	e.done = make(chan struct{})
	//fmt.Println("del events reset")
}

func (e *delEvents) stop() {
	close(e.done)
	//fmt.Println("del events stopped")
}
