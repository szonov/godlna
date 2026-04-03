//go:build linux

package fswatcher

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"
)

const eventsBufferSize = 4096 * (unix.SizeofInotifyEvent + unix.NAME_MAX + 1)
const inotifyMask = unix.IN_CREATE | unix.IN_DELETE | unix.IN_MOVED_FROM | unix.IN_MOVED_TO | unix.IN_CLOSE_WRITE

const (
	stateStopped int32 = iota
	stateRunning
	stateStopping
)

type inotify struct {
	roots []string      // list of root directories for watching
	state int32         // 0 - stopped, 1 - running, 2 - stopping
	fd    int           // inotify file descriptor
	epfd  int           // epoll file descriptor
	done  chan struct{} // signals watcher shutdown

	ignoreFn     IgnoreFn     // callback define is file/directory should be excluded from watching
	eventHandler EventHandler // event handler callback.
	errorHandler ErrorHandler // callback handle errors during watching

	delEvents *delEvents
	mvEvents  *mvEvents

	watchMap   *watchMap
	stopDoneCh chan struct{}
}

func newDriver() (driver, error) {
	w := &inotify{
		roots: make([]string, 0),
		state: stateStopped,
	}

	w.watchMap = newWatchMap()
	w.delEvents = newDeleteEvents(w.handleDeleteEvent)
	w.mvEvents = newMvEvents(w.handleMoveEvent)

	return w, nil
}

func (w *inotify) addDirectory(dir string) error {
	absPath, err := validatedAddDir(dir, w.roots)
	if err != nil {
		return fmt.Errorf("*inotify:addDirectory: %w", err)
	}
	w.roots = append(w.roots, absPath)
	return nil
}

func (w *inotify) start() error {
	if !atomic.CompareAndSwapInt32(&w.state, stateStopped, stateRunning) {
		return fmt.Errorf("*inotify:start: watcher is already running")
	}

	if len(w.roots) == 0 {
		return errors.New("*inotify:start: at least one directory should be defined")
	}

	if err := w.preStart(); err != nil {
		atomic.StoreInt32(&w.state, stateStopped)
		return err
	}

	w.sendEvent(Event{Op: WalkStart})

	for _, root := range w.roots {
		if err := w.walkStartingAt(root); err != nil {
			w.shutdown()
			return err
		}
	}

	w.sendEvent(Event{Op: WalkComplete})

	go w.readInotifyEvents()

	return nil
}

func (w *inotify) stop() error {
	if !atomic.CompareAndSwapInt32(&w.state, stateRunning, stateStopping) {
		fmt.Println("watcher is already stopping")
		return nil
	}

	// Signal shutdown
	close(w.done)

	// Wait for complete shutdown
	<-w.stopDoneCh

	return nil

}

// withEventHandler sets the event handler callback.
func (w *inotify) withEventHandler(fn EventHandler) {
	w.eventHandler = fn
}

// withErrorHandler sets the error handler callback.
func (w *inotify) withErrorHandler(fn ErrorHandler) {
	w.errorHandler = fn
}

// withIgnoreFn sets callback for detecting ignored file or directory
func (w *inotify) withIgnoreFn(fn IgnoreFn) {
	w.ignoreFn = fn
}

// shouldIgnore check if file/dir basename should be excluded from eventing
func (w *inotify) shouldIgnore(absPath string, isDir bool) bool {
	if w.ignoreFn == nil {
		return false
	}
	return w.ignoreFn(absPath, isDir)
}

func (w *inotify) sendEvent(e Event) {
	if w.eventHandler != nil {
		w.eventHandler(e)
	}
}

func (w *inotify) sendError(err error) {
	if w.errorHandler != nil {
		w.errorHandler(err)
	}
}

func (w *inotify) preStart() error {

	// Initialize inotify with non-blocking flag
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return fmt.Errorf("inotify_init1 failed: %w", err)
	}

	// Create epoll instance for efficient waiting
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		_ = unix.Close(fd)
		return fmt.Errorf("epoll_create1 failed: %w", err)
	}

	// Add inotify fd to epoll
	event := unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	}

	if err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		_ = unix.Close(epfd)
		_ = unix.Close(fd)
		return fmt.Errorf("epoll_ctl_add failed: %w", err)
	}

	w.fd = fd
	w.epfd = epfd

	w.done = make(chan struct{})
	w.stopDoneCh = make(chan struct{})

	w.watchMap.reset()
	w.delEvents.reset()
	w.mvEvents.reset()

	return nil
}

func (w *inotify) walkStartingAt(rootPath string) error {

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

		if isDir {
			wd, err := unix.InotifyAddWatch(w.fd, walkPath, inotifyMask)
			if err != nil {
				return fmt.Errorf("failed to add '%s' to inotify instance: %w", walkPath, err)
			}
			w.watchMap.add(uint32(wd), walkPath)
		}

		w.sendEvent(Event{Op: Index, Name: walkPath, IsDir: isDir})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory '%s': %w", rootPath, err)
	}

	return nil
}

func (w *inotify) readInotifyEvents() {
	defer w.shutdown()

	buf := make([]byte, eventsBufferSize)

	events := make([]unix.EpollEvent, 1)

	for {
		select {
		case <-w.done:
			return
		default:
		}

		// Wait for events with timeout for checking done channel
		n, err := unix.EpollWait(w.epfd, events, 100)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			w.sendError(fmt.Errorf("epoll_wait failed: %w", err))
			return
		}

		// no new events
		if n == 0 {
			continue
		}

		// Read events from inotify
		bytesRead, err := unix.Read(w.fd, buf)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
				continue
			}
			w.sendError(fmt.Errorf("read failed: %w", err))
			return
		}

		if bytesRead < unix.SizeofInotifyEvent {
			continue
		}

		buffer := buf[:bytesRead]

		offset := 0

		for offset < len(buffer) {
			if offset+unix.SizeofInotifyEvent > len(buffer) {
				break
			}

			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buffer[offset]))
			offset += unix.SizeofInotifyEvent

			// Get the name if present
			var name string
			if raw.Len > 0 {
				nameBytes := buffer[offset : offset+int(raw.Len)]
				// Find null terminator
				for i, b := range nameBytes {
					if b == 0 {
						name = string(nameBytes[:i])
						break
					}
				}
				offset += int(raw.Len)
			}
			w.handleInotifyEvent(name, uint32(raw.Wd), raw.Mask, raw.Cookie)
		}
	}
}

func (w *inotify) handleInotifyEvent(baseName string, wd uint32, mask, cookie uint32) {

	isDir := mask&unix.IN_ISDIR == unix.IN_ISDIR

	var absPath string
	if parent, ok := w.watchMap.path(wd); ok {
		absPath = path.Join(parent, baseName)
	} else {
		w.sendError(fmt.Errorf("failed to find path for wd: %d", wd))
		return
	}

	if w.shouldIgnore(absPath, isDir) {
		return
	}

	if mask&unix.IN_CREATE == unix.IN_CREATE {
		if isDir {
			// using w.addDirsStartingAt(..) we handle case for mkdir -p 1/2/3,
			// since originally we got only one event for creating '1' directory
			if err := w.walkStartingAt(absPath); err != nil {
				w.sendError(fmt.Errorf("failed to add path '%s': %w", absPath, err))
			}
		}
		if strings.HasSuffix(absPath, "100") {
			w.watchMap.debug()
		}
	} else if mask&unix.IN_IGNORED == unix.IN_IGNORED {
		fmt.Printf("[IN_IGNORED] for wd: %d '%s'\n", wd, absPath)
	} else if mask&unix.IN_DELETE == unix.IN_DELETE {
		w.delEvents.add(absPath, isDir)
		if isDir {
			// we do not need to run unix.InotifyRmWatch(...) for folder and subfolders,
			// since it is done by linux kernel automatically
			// we just clear remembered wd and path
			w.watchMap.rmByPath(absPath)
		}
	} else if mask&unix.IN_MOVED_FROM == unix.IN_MOVED_FROM {
		w.mvEvents.addMvFrom(cookie, absPath, isDir)
	} else if mask&unix.IN_MOVED_TO == unix.IN_MOVED_TO {
		w.mvEvents.addMvTo(cookie, absPath, isDir)
	} else if !isDir && mask&unix.IN_CLOSE_WRITE == unix.IN_CLOSE_WRITE {
		w.sendEvent(Event{Op: Index, Name: absPath, IsDir: isDir})
	}
}

func (w *inotify) handleDeleteEvent(name string, isDir bool) {
	w.sendEvent(Event{Op: Remove, Name: name, IsDir: isDir})
}

func (w *inotify) handleMoveEvent(op Op, oldName, newName string, isDir bool) {
	if op == Rename {
		if isDir {
			w.watchMap.rename(oldName, newName)
		}
		w.sendEvent(Event{Op: op, Name: newName, IsDir: isDir, RenamedFrom: oldName})
		return
	}

	if op == Remove {
		if isDir {
			wds := w.watchMap.rmByPathRecursive(oldName)
			for _, wd := range wds {
				if _, err := unix.InotifyRmWatch(w.fd, wd); err != nil && !errors.Is(err, unix.EINVAL) {
					w.sendError(fmt.Errorf("inotify_rm_watch failed for wd %d: %w", wd, err))
				}
			}
		}
		w.sendEvent(Event{Op: op, Name: oldName, IsDir: isDir})
		return
	}

	if op == Index {
		if isDir {
			if err := w.walkStartingAt(newName); err != nil {
				w.sendError(fmt.Errorf("move_to indexing failed for '%s': %w", newName, err))
			}
		} else {
			w.sendEvent(Event{Op: op, Name: newName, IsDir: isDir})
		}
		return
	}
}

func (w *inotify) shutdown() {
	//fmt.Printf("shutdown ...\n")
	var err error

	// detach inotify file descriptor from epoll
	if err = unix.EpollCtl(w.epfd, unix.EPOLL_CTL_DEL, w.fd, nil); err != nil {
		w.sendError(fmt.Errorf("epoll_ctl_del(%d) failed: %w", w.epfd, err))
	}

	// close inotify file descriptor
	if err = unix.Close(w.fd); err != nil {
		w.sendError(fmt.Errorf("close inotify fd (%d) failed: %w", w.fd, err))
	}

	// close epoll file descriptor
	if err = unix.Close(w.epfd); err != nil {
		w.sendError(fmt.Errorf("close epoll fd (%d) failed: %w", w.epfd, err))
	}

	w.delEvents.stop()
	w.mvEvents.stop()

	atomic.StoreInt32(&w.state, stateStopped)

	close(w.stopDoneCh)
}
