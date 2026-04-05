package fswatcher

import (
	"fmt"
)

// Op describes a set of file operations.
type Op uint32

// The operations watcher can trigger
const (
	// Index The file or directory must be indexed (appear in the directory being watched)
	Index Op = 1 << iota

	// Remove The path was removed from watched directory (deleted or moved to another directory).
	Remove

	// Rename The path was renamed. Both source and destination a placed in watched directory.
	Rename

	// WalkStart When the watcher starts, it traverses the tree of the directory being viewed
	// (an Index event is sent for each item), and before this traversal is started, a WalkStart event is sent.
	WalkStart

	// WalkComplete When the watcher starts, it traverses the tree of the directory being viewed
	// (an Index event is sent for each item), and after this traversal is completed, a WalkComplete event is sent.
	WalkComplete
)

func (op Op) String() string {
	switch op {
	case Index:
		return "INDEX"
	case Remove:
		return "REMOVE"
	case Rename:
		return "RENAME"
	case WalkStart:
		return "WALK_START"
	case WalkComplete:
		return "WALK_COMPLETE"
	default:
		return "UNKNOWN"
	}
}

type Event struct {
	// Op operation that triggered the event.
	Op Op

	// Name is absolute path to file or directory
	Name string

	// IsDir flag indicating that the event is for a file or directory
	IsDir bool

	// RenamedFrom is source path in Rename operation
	// For example "mv /tmp/oldfile.txt /tmp/newfile.txt" will emit:
	//
	//   Event{Op: Rename, IsDir: false, Name: "/tmp/newfile.txt", RenamedFrom: "/tmp/oldfile.txt"}
	RenamedFrom string
}

func (e Event) String() string {
	if e.Op == WalkComplete || e.Op == WalkStart {
		return fmt.Sprintf("[%s]", e.Op)
	}
	var typ string
	if e.IsDir {
		typ = "DIR"
	} else {
		typ = "FILE"
	}
	if e.Op == Rename {
		return fmt.Sprintf("[%s:%s] '%s' -> '%s'", e.Op, typ, e.RenamedFrom, e.Name)
	}
	return fmt.Sprintf("[%s:%s] '%s'", e.Op, typ, e.Name)
}

// EventHandler is a callback function that handles file system events.
type EventHandler func(event Event)

// ErrorHandler is a callback function that handles errors during watching.
type ErrorHandler func(err error)

// IgnoreFn is a callback function that define should be file or directory excluded from eventing
type IgnoreFn func(absPath string, isDir bool) bool

type Watcher struct {
	d driver
}

// New creates a new Watcher.
func New(dirs ...string) (*Watcher, error) {
	d, err := newDriver()
	if err != nil {
		return nil, err
	}
	w := &Watcher{d: d}
	for _, dir := range dirs {
		if err = w.Add(dir); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// Add adds directory to watch.
// NOTICE: directory should be added before watcher starts
func (w *Watcher) Add(dir string) error {
	return w.d.addDirectory(dir)
}

// WatchList returns all paths explicitly added with [Watcher.Add]
func (w *Watcher) WatchList() []string {
	return w.d.watchList()
}

// Start starts the watcher
func (w *Watcher) Start() error {
	return w.d.start()
}

// Stop stops watching for filesystems events and free resources
func (w *Watcher) Stop() error {
	return w.d.stop()
}

// WithEventHandler sets the event handler callback.
func (w *Watcher) WithEventHandler(fn EventHandler) {
	w.d.withEventHandler(fn)
}

// WithErrorHandler sets the error handler callback.
func (w *Watcher) WithErrorHandler(fn ErrorHandler) {
	w.d.withErrorHandler(fn)
}

// WithIgnoreFn sets callback for detecting ignored file or directory
func (w *Watcher) WithIgnoreFn(handler IgnoreFn) {
	w.d.withIgnoreFn(handler)
}

type driver interface {
	addDirectory(string) error
	start() error
	stop() error
	withEventHandler(EventHandler)
	withErrorHandler(ErrorHandler)
	withIgnoreFn(IgnoreFn)
	watchList() []string
}
