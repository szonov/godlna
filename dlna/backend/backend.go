package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/szonov/godlna/pkg/fswatcher"
)

var videoExtensions = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

func ignoreFn(name string, isDir bool) bool {
	if isDir {
		return strings.HasSuffix(name, "/@eaDir") || strings.Contains(name, "/@eaDir/")
	}
	if strings.Contains(name, "/@eaDir/") {
		return true
	}
	fileExt := strings.ToLower(filepath.Ext(name))
	for _, ext := range videoExtensions {
		if fileExt == ext {
			return false
		}
	}
	return true
}

var (
	ErrNoRows = errors.New("backend: no rows in result set")
)

type ObjectType int

const (
	ObjectFolder ObjectType = iota
	ObjectVideo
)

type Object struct {
	ID         int
	Path       string
	Typ        ObjectType
	Format     string
	FileSize   int64
	VideoCodec string
	AudioCodec string
	Width      int
	Height     int
	Channels   int
	Bitrate    int
	Frequency  int
	Duration   int64
	Bookmark   sql.NullInt64
	Date       int64
	Online     bool
	IsDirty    bool
}

func (o *Object) ThumbPath() string {
	if o.Typ == ObjectVideo {
		return thumbnailFile(o.Path)
	}
	return ""
}

func (o *Object) Title() string {
	filename := filepath.Base(o.Path)
	if o.Typ == ObjectFolder {
		return filename
	}
	ext := filepath.Ext(filename)
	return filename[0 : len(filename)-len(ext)]
}

type ObjectSearchFilter struct {
	Id               int
	Limit            int
	Offset           int
	ParentPath       string
	OwnPaths         []string
	Dirty            sql.NullBool
	WithTotalMatches bool
	// LastVisitedId used for loop dirty objects,
	// if passed valid value, order of search results should be changed to 'ORDER BY id'
	LastVisitedId sql.NullInt64
}

type ObjectSearchResponse struct {
	Items        []*Object
	TotalMatches int
}

type Backend struct {
	roots     []string
	d         DatabaseDriver
	w         *fswatcher.Watcher
	done      chan struct{}
	dirtyFlag uint32
}

func NewBackend(roots []string, d DatabaseDriver) (*Backend, error) {
	b := &Backend{d: d}

	watcher, err := fswatcher.New(roots...)
	if err != nil {
		return nil, err
	}

	watcher.WithErrorHandler(b.onError)
	watcher.WithEventHandler(b.onWatcherEvent)
	watcher.WithIgnoreFn(ignoreFn)

	b.w = watcher

	// add absolute path for roots after watcher created,
	// now it's converted to absolute paths
	b.roots = watcher.WatchList()

	return b, nil
}

func (b *Backend) Start() error {
	b.done = make(chan struct{})
	return b.w.Start()
}

func (b *Backend) Stop() error {
	close(b.done)
	return b.w.Stop()
}

func (b *Backend) onError(err error) {
	if err != nil {
		slog.Error(err.Error())
	}
}

func (b *Backend) onWatcherEvent(e fswatcher.Event) {
	slog.Debug("EVENT", "e", e.String())
	var err error
	switch e.Op {
	case fswatcher.WalkStart:
		err = b.d.AllObjectsToOffline()
	case fswatcher.WalkComplete:
		err = b.d.DeleteOfflineObjects()
		if err == nil {
			go func() {
				b.ReindexDirty()
				for {
					select {
					case <-b.done:
						return
					case <-time.After(30 * time.Second):
						b.ReindexDirty()
					}
				}
			}()

		}
	case fswatcher.Index:
		err = b.d.Index(e.IsDir, e.Name)
		// setup dirtyFlag when something new
		atomic.StoreUint32(&b.dirtyFlag, 1)
	case fswatcher.Remove:
		err = b.d.Remove(e.IsDir, e.Name)
	case fswatcher.Rename:
		err = b.d.Rename(e.IsDir, e.RenamedFrom, e.Name)
	}
	b.onError(err)
}

func (b *Backend) Object(id int) (*Object, error) {
	if id <= 0 { // root object
		return &Object{
			ID:     0,
			Path:   "Video Root",
			Typ:    ObjectFolder,
			Online: true,
		}, nil
	}

	filter := ObjectSearchFilter{
		Id:    id,
		Dirty: sql.NullBool{Bool: false, Valid: true},
	}

	res, err := b.d.GetObjects(filter)
	if err != nil {
		return nil, err
	}

	if len(res.Items) > 0 {
		return res.Items[0], err
	}

	return nil, ErrNoRows
}

func (b *Backend) Children(o *Object, limit int, offset int) (*ObjectSearchResponse, error) {
	filter := ObjectSearchFilter{
		ParentPath:       o.Path,
		Dirty:            sql.NullBool{Bool: false, Valid: true},
		Limit:            limit,
		Offset:           offset,
		WithTotalMatches: true,
	}
	if o.ID <= 0 { // root children
		switch len(b.roots) {
		case 0: // not possible in normal usage
			return &ObjectSearchResponse{Items: make([]*Object, 0)}, nil

		case 1: // single - root, response with children of root path
			filter.ParentPath = b.roots[0]

		default: // multi-root, response with root folders
			filter.ParentPath = ""
			filter.OwnPaths = b.roots
		}
	}

	return b.d.GetObjects(filter)
}

func (b *Backend) ParentId(o *Object) (int, error) {
	if o.ID <= 0 {
		return -1, nil
	}

	if len(b.roots) > 1 { // multi-root, check if object is one or root folders
		for _, root := range b.roots {
			if o.Path == root {
				return 0, nil
			}
		}
	}

	res, err := b.d.GetObjects(ObjectSearchFilter{
		OwnPaths: []string{filepath.Dir(o.Path)},
		Dirty:    sql.NullBool{Bool: false, Valid: true},
		Limit:    1,
	})

	if err != nil || len(res.Items) == 0 {
		return 0, fmt.Errorf("(backend.ParentId) no object found: %w", err)
	}

	parent := res.Items[0]

	if len(b.roots) == 1 && parent.Path == b.roots[0] { // in single root mode found root folder
		return 0, nil
	}

	return parent.ID, nil
}

// SetBookmark for given ID sets bookmark time in milliseconds
func (b *Backend) SetBookmark(id int, bookmark int64) error {
	if bookmark < 0 {
		return nil
	}
	if id <= 0 {
		return ErrNoRows
	}

	res, err := b.d.GetObjects(ObjectSearchFilter{Id: id})
	if err != nil || len(res.Items) == 0 {
		return ErrNoRows
	}

	o := res.Items[0]
	bmi := &BookmarkInfo{
		Bookmark: sql.NullInt64{Int64: bookmark, Valid: true},
	}

	if o.Bookmark == bmi.Bookmark {
		// nothing changed
		return nil
	}

	// Store to database
	if err := b.d.UpdateObject(o, nil, bmi); err != nil {
		return err
	}

	// Create thumbnail
	if err := makeThumbnail(o.Path, o.Duration, bmi.Bookmark); err != nil {
		return err
	}

	// Store to cache file
	if err := SetBookmarkInfo(o.Path, bmi); err != nil {
		return err
	}

	return nil
}

func (b *Backend) Reindex(o *Object) error {
	if o == nil || o.ID <= 0 || o.Typ != ObjectVideo {
		return nil
	}

	videoInfo, err := GetVideoInfo(o.Path)
	if err != nil {
		return err
	}

	bmi, err := GetBookmarkInfo(o.Path)
	if err != nil {
		return err
	}

	if err := b.d.UpdateObject(o, videoInfo, bmi); err != nil {
		return err
	}

	if !isThumbnailExists(o.Path) {
		return makeThumbnail(o.Path, o.Duration, bmi.Bookmark)
	}

	return nil
}

func (b *Backend) ReindexDirty() {

	if !atomic.CompareAndSwapUint32(&b.dirtyFlag, 1, 0) {
		slog.Debug(">>>>>>> ReindexDirty - skip, flag is 0")
		return
	}

	slog.Debug(">>>>>>> ReindexDirty")

	filter := ObjectSearchFilter{
		Dirty:         sql.NullBool{Bool: true, Valid: true},
		LastVisitedId: sql.NullInt64{Int64: 0, Valid: true},
		Limit:         10,
	}
	for {
		res, err := b.d.GetObjects(filter)

		if err != nil {
			slog.Error("ReindexDirty :: GetObjects", "err", err)
			return
		}

		if len(res.Items) == 0 {
			return
		}

		for _, o := range res.Items {
			select {
			case <-b.done:
				return
			default:
			}
			filter.LastVisitedId.Int64 = int64(o.ID)

			if err := b.Reindex(o); err != nil {
				slog.Error("ReindexDirty :: Reindex", "err", err, "o", o)
			} else {
				slog.Debug("ReindexDirty", "id", o.ID, "path", o.Path)
			}
		}
	}
}
