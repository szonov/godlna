package backend

type DatabaseDriver interface {
	GetObjects(filter ObjectSearchFilter) (result *ObjectSearchResponse, err error)
	UpdateObject(item *Object, videoInfo *VideoInfo, bookmarkInfo *BookmarkInfo) (err error)

	AllObjectsToOffline() (err error)
	DeleteOfflineObjects() (err error)

	Index(isDir bool, fullPath string) (err error)
	Remove(isDir bool, fullPath string) (err error)
	Rename(isDir bool, oldFullPath string, newFullPath string) (err error)
}
