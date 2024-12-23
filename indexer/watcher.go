package indexer

import (
	"github.com/rjeczalik/notify"
	"log/slog"
	"strings"
)

type WatcherCallback func(path string)

type Watcher struct {
	dir      string
	done     chan struct{}
	callback WatcherCallback
}

func NewWatcher(watchDir string, callback WatcherCallback) *Watcher {
	return &Watcher{
		dir:      watchDir,
		callback: callback,
	}
}

func (w *Watcher) Start() {
	go w.StartAndListen()
}

func (w *Watcher) StartAndListen() {
	w.done = make(chan struct{})
	w.listen()
}

func (w *Watcher) Stop() {
	if w.done == nil {
		return
	}
	close(w.done)
}

func (w *Watcher) listen() {
	notifyChan := make(chan notify.EventInfo, 100)
	watchDir := strings.TrimSuffix(w.dir, "/") + "/..."
	if err := notify.Watch(watchDir, notifyChan, notify.All); err != nil {
		slog.Error("watch failed", "err", err.Error(), "dir", w.dir)
		close(w.done)
		close(notifyChan)
		return
	}
	for {
		select {
		case ei := <-notifyChan:
			w.callback(ei.Path())
		case <-w.done:
			w.done = nil
			notify.Stop(notifyChan)
			return
		default:
		}
	}
}
