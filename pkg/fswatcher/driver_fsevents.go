//go:build darwin

package fswatcher

/*
#cgo LDFLAGS: -framework CoreServices
#include <CoreServices/CoreServices.h>
#include <dispatch/dispatch.h>

// Forward declaration of the Go callback.
extern void fseventsCallback(
    uintptr_t id,
    size_t numEvents,
    char **paths,
	FSEventStreamEventFlags* flags,
	FSEventStreamEventId* ids
);

// C wrapper that FSEventStreamCreate receives as callback.
static void cfCallback(
    ConstFSEventStreamRef stream,
    void *info,
    size_t numEvents,
    void *eventPaths,
    const FSEventStreamEventFlags eventFlags[],
    const FSEventStreamEventId eventIds[])
{
    uintptr_t id = (uintptr_t)info;
    fseventsCallback(id, numEvents, (char **)eventPaths,(FSEventStreamEventFlags*)eventFlags, (FSEventStreamEventId*)eventIds);
}

static void dispatch_release_queue(dispatch_queue_t q) {
    dispatch_release(q);
}

// createCFString wraps CFStringCreateWithCString, returning void* for the
// CFArray element storage (avoids unsafe.Pointer vet warnings in Go).
static void *createCFString(const char *s) {
    return (void *)CFStringCreateWithCString(kCFAllocatorDefault, s, kCFStringEncodingUTF8);
}

static FSEventStreamRef createStream(
    CFArrayRef paths,
    uintptr_t id,
    double latency,
    uint32_t flags)
{
    FSEventStreamContext ctx = {0, (void *)id, NULL, NULL, NULL};
    return FSEventStreamCreate(
        NULL,
        cfCallback,
        &ctx,
        paths,
        kFSEventStreamEventIdSinceNow,
        latency,
        flags
    );
}

// Wrappers to avoid unsafe.Pointer↔uintptr round-trips in Go.
static void streamStop(FSEventStreamRef s)      { FSEventStreamStop(s); }
static void streamInvalidate(FSEventStreamRef s) { FSEventStreamInvalidate(s); }
static void streamRelease(FSEventStreamRef s)    { FSEventStreamRelease(s); }
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

type renameItem struct {
	id       uint64
	isDir    bool
	fullPath string
	done     chan struct{}
}

type fsevents struct {
	roots []string // list of root directories for watching

	ignoreFn     IgnoreFn     // callback define is file/directory should be excluded from watching
	eventHandler EventHandler // event handler callback.
	errorHandler ErrorHandler // callback handle errors during watching

	delEvents *delEvents

	latency        float64
	stream         unsafe.Pointer // FSEventStreamRef
	id             uintptr
	lastRenameItem *renameItem

	done chan struct{} // signals watcher shutdown
}

func (e *renameItem) close() {
	if e.done == nil {
		return
	}
	select {
	case <-e.done:
	default:
		close(e.done)
	}
}

func newDriver() (driver, error) {
	w := &fsevents{
		roots:          make([]string, 0),
		latency:        1, // 1 second
		lastRenameItem: &renameItem{},
	}
	w.delEvents = newDeleteEvents(w.handleDeleteEvent)

	return w, nil
}

func (w *fsevents) addDirectory(dir string) error {
	absPath, err := validatedAddDir(dir, w.roots)
	if err != nil {
		return fmt.Errorf("*inotify:addDirectory: %w", err)
	}
	w.roots = append(w.roots, absPath)
	return nil
}

// withEventHandler sets the event handler callback.
func (w *fsevents) withEventHandler(fn EventHandler) {
	w.eventHandler = fn
}

// withErrorHandler sets the error handler callback.
func (w *fsevents) withErrorHandler(fn ErrorHandler) {
	w.errorHandler = fn
}

// withIgnoreFn sets callback for detecting ignored file or directory
func (w *fsevents) withIgnoreFn(fn IgnoreFn) {
	w.ignoreFn = fn
}

// watchList returns all paths explicitly added with [fsevents.addDirectory]
func (w *fsevents) watchList() []string {
	return w.roots
}

// shouldIgnore check if file/dir basename should be excluded from eventing
func (w *fsevents) shouldIgnore(absPath string, isDir bool) bool {
	if w.ignoreFn == nil {
		return false
	}
	return w.ignoreFn(absPath, isDir)
}

func (w *fsevents) sendEvent(e Event) {
	if w.eventHandler != nil {
		w.eventHandler(e)
	}
}

func (w *fsevents) sendError(err error) {
	if w.errorHandler != nil {
		w.errorHandler(err)
	}
}

func (w *fsevents) walkStartingAt(rootPath string) error {

	// Walk and add all subdirectories
	err := filepath.Walk(rootPath, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		isDir := info.IsDir()

		if w.shouldIgnore(walkPath, isDir) {
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}

		w.sendEvent(Event{Op: Index, Name: walkPath, IsDir: isDir})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory '%s': %w", rootPath, err)
	}

	return nil
}

// Global registry — maps integer IDs to *fsevents so we never pass
// Go pointers through C (satisfies cgo pointer rules).
var (
	mu       sync.Mutex
	registry = map[uintptr]*fsevents{}
	nextID   uintptr
)

func register(w *fsevents) uintptr {
	mu.Lock()
	nextID++
	id := nextID
	registry[id] = w
	mu.Unlock()
	return id
}

func unregister(id uintptr) {
	mu.Lock()
	delete(registry, id)
	mu.Unlock()
}

func lookup(id uintptr) *fsevents {
	mu.Lock()
	w := registry[id]
	mu.Unlock()
	return w
}

//export fseventsCallback
func fseventsCallback(id C.uintptr_t, numEvents C.size_t, cPaths **C.char, cFlags *C.FSEventStreamEventFlags, cIds *C.FSEventStreamEventId) {
	w := lookup(uintptr(id))
	if w == nil {
		return
	}
	n := int(numEvents)
	slice := unsafe.Slice(cPaths, n)
	flagsSlice := unsafe.Slice(cFlags, n)
	idsSlice := unsafe.Slice(cIds, n)

	for i := 0; i < n; i++ {
		w.handleFsEvent(uint64(idsSlice[i]), C.GoString(slice[i]), uint32(flagsSlice[i]))
	}
	//fmt.Printf("START SLEEPING\n")
	//time.Sleep(10 * time.Second)
	//fmt.Printf("STOP SLEEPING\n")
}

// Start begins watching.
// The FSEvents stream is scheduled on a serial dispatch queue.
func (w *fsevents) start() error {
	if len(w.roots) == 0 {
		return errors.New("*fsevents:start: at least one directory should be defined")
	}

	w.delEvents.reset()

	w.sendEvent(Event{Op: WalkStart})

	for _, root := range w.roots {
		if err := w.walkStartingAt(root); err != nil {
			return err
		}
	}

	w.sendEvent(Event{Op: WalkComplete})

	w.id = register(w)

	// Build CFArray of paths.
	cPaths := make([]unsafe.Pointer, len(w.roots))
	for i, p := range w.roots {
		cs := C.CString(p)
		cPaths[i] = C.createCFString(cs)
		C.free(unsafe.Pointer(cs))
	}
	cfArr := C.CFArrayCreate(
		C.kCFAllocatorDefault,
		(*unsafe.Pointer)(unsafe.Pointer(&cPaths[0])),
		C.CFIndex(len(cPaths)),
		&C.kCFTypeArrayCallBacks,
	)
	defer C.CFRelease(C.CFTypeRef(cfArr))
	for _, p := range cPaths {
		C.CFRelease(C.CFTypeRef(p))
	}

	flags := C.uint32_t(C.kFSEventStreamCreateFlagNoDefer | C.kFSEventStreamCreateFlagFileEvents)
	stream := C.createStream(cfArr, C.uintptr_t(w.id), C.double(w.latency), flags)
	if stream == nil {
		unregister(w.id)
		return fmt.Errorf("*fsevents: FSEventStreamCreate failed")
	}
	w.stream = unsafe.Pointer(stream)

	label := C.CString("com.godlna.fsevents")
	queue := C.dispatch_queue_create(label, nil)
	C.free(unsafe.Pointer(label))
	C.FSEventStreamSetDispatchQueue(stream, queue)
	C.dispatch_release_queue(queue)

	if C.FSEventStreamStart(stream) == C.Boolean(0) {
		C.FSEventStreamInvalidate(stream)
		C.FSEventStreamRelease(stream)
		w.stream = nil
		unregister(w.id)
		return fmt.Errorf("*fsevents: FSEventStreamStart failed")
	}

	w.done = make(chan struct{})
	return nil
}

// Stop tears down the FSEvents stream and closes the Events channel.
// It is safe to call multiple times.
func (w *fsevents) stop() error {
	if w.stream == nil {
		return nil
	}
	stream := (C.FSEventStreamRef)(w.stream)
	w.stream = nil
	C.streamStop(stream)
	C.streamInvalidate(stream)
	C.streamRelease(stream)

	close(w.done)
	w.delEvents.stop()

	unregister(w.id)

	return nil
}

func (w *fsevents) handleFsEvent(id uint64, fullPath string, flags uint32) {
	// only handle directory or file events (no symlinks, no hardlinks)
	isDir := flags&FSEventStreamEventFlagItemIsDir == FSEventStreamEventFlagItemIsDir
	isFile := flags&FSEventStreamEventFlagItemIsFile == FSEventStreamEventFlagItemIsFile

	w.lastRenameItem.close()

	if !isDir && !isFile {
		return
	}

	if w.shouldIgnore(fullPath, isDir) {
		if w.lastRenameItem.id > 0 {
			w.handleLastRenameEvent(w.lastRenameItem.fullPath, w.lastRenameItem.isDir)
		}
		return
	}

	isRename := flags&FSEventStreamEventFlagItemRenamed == FSEventStreamEventFlagItemRenamed
	isRemove := flags&FSEventStreamEventFlagItemRemoved == FSEventStreamEventFlagItemRemoved
	isCreate := flags&FSEventStreamEventFlagItemCreated == FSEventStreamEventFlagItemCreated
	isClone := flags&FSEventStreamEventFlagItemCloned == FSEventStreamEventFlagItemCloned
	isModify := flags&FSEventStreamEventFlagItemModified == FSEventStreamEventFlagItemModified

	//fmt.Printf(" ------ DIR: [%d] [%s] (%s) [%d] (rename=%v)\n", id, fullPath, ParseDarwinEventFlags(flags), flags, isRename)

	if isRename && w.lastRenameItem.id+1 == id && isDir == w.lastRenameItem.isDir {
		// paired rename event
		w.sendEvent(Event{Op: Rename, IsDir: isDir, Name: fullPath, RenamedFrom: w.lastRenameItem.fullPath})
		w.lastRenameItem.id = 0
		return
	}

	if w.lastRenameItem.id > 0 {
		w.handleLastRenameEvent(w.lastRenameItem.fullPath, w.lastRenameItem.isDir)
	}

	if isRemove {
		w.delEvents.add(fullPath, isDir)
		//w.sendEvent(Event{Op: Remove, IsDir: isDir, Name: fullPath})
		return
	}

	if isRename {
		w.updateLastRenameEvent(id, fullPath, isDir)
		return
	}

	if isCreate {
		if isDir || isModify {
			w.sendEvent(Event{Op: Index, IsDir: isDir, Name: fullPath})
		}
		return
	}

	if isModify || isClone {
		w.sendEvent(Event{Op: Index, IsDir: isDir, Name: fullPath})
		return
	}

	w.sendError(fmt.Errorf("*fsevents.handleFsEvent unhandled case '%s' (%s)", fullPath, ParseDarwinEventFlags(flags)))
}

func (w *fsevents) handleDeleteEvent(name string, isDir bool) {
	w.sendEvent(Event{Op: Remove, Name: name, IsDir: isDir})
}

func (w *fsevents) handleLastRenameEvent(fullPath string, isDir bool) {
	w.lastRenameItem.id = 0

	// file or directory does not exist
	if _, err := os.Stat(fullPath); err != nil {
		w.sendEvent(Event{Op: Remove, IsDir: isDir, Name: fullPath})
		return
	}

	// file or directory does exist
	if isDir {
		if err := w.walkStartingAt(fullPath); err != nil {
			w.sendError(fmt.Errorf("lazy walk error: %w", err))
		}
	} else {
		w.sendEvent(Event{Op: Index, IsDir: isDir, Name: fullPath})
	}
}

func (w *fsevents) updateLastRenameEvent(id uint64, fullPath string, isDir bool) {
	done := make(chan struct{})
	w.lastRenameItem.id = id
	w.lastRenameItem.fullPath = fullPath
	w.lastRenameItem.isDir = isDir
	w.lastRenameItem.done = done
	go func() {
		select {
		case <-done:
		case <-w.done:
		case <-time.After(1 * time.Second):
			//red(">>> TIMEOUT HAPPENS FOR %v\n", w.lastRenameItem)
			w.handleLastRenameEvent(fullPath, isDir)
		}
	}()
}
