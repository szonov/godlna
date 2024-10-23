package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/types"
	"log/slog"
	"os"
	"path"
	"strings"
)

const (
	TypeFolder int = 1
	TypeVideo  int = 2
	TypeStream int = 3
)

var (
	// mediaDir is full path to media directory
	mediaDir string
	// cacheDir is full path to cache directory (where database and cached thumbnails stored)
	cacheDir string
	// db is instance of *sql.DB
	db *sql.DB
)

type VideoMeta struct {
	Resolution string          `json:"r"`
	Channels   uint32          `json:"c"`
	SampleRate uint32          `json:"s"`
	BitRate    uint32          `json:"b"`
	Duration   *types.Duration `json:"d"`
	Format     string          `json:"f"`
	VideoCodec string          `json:"v"`
	AudioCodec string          `json:"a"`
}

func (v *VideoMeta) Value() (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

type StreamMeta struct {
	Profile  string   `json:"Profile,omitempty"`
	MimeType string   `json:"MimeType,omitempty"`
	Command  []string `json:"Command"`
}

func (s *StreamMeta) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

type Object struct {
	ObjectID  string
	ParentID  string
	Path      string
	Type      int
	Version   int
	Timestamp *types.NullableNumber
	Size      uint64
	Bookmark  *types.Duration
	Meta      interface{}
}

// getObjectCacheDir returns path to directory where all object's cached resources stored
func getObjectCacheDir(objectID string) string {
	return path.Join(cacheDir, "thumbs", strings.Replace(objectID, "$", "/", -1))
}

func removeObjectCache(objectID string) {
	_ = os.RemoveAll(getObjectCacheDir(objectID))
}

func GetObject(objectID string, upToDate ...bool) *Object {
	if objectID == "" {
		return nil
	}
	if len(upToDate) > 0 && upToDate[0] {
		EnsureObjectIsUpToDate(objectID)
	}
	o := filteredObjects("OBJECT_ID", objectID, 1, 0)
	if len(o) > 0 {
		return o[0]
	}
	return nil
}

func (o *Object) ChildCount() uint64 {
	var totalCount uint64
	q := `SELECT COUNT(*) FROM OBJECTS WHERE PARENT_ID = ?`
	if err := db.QueryRow(q, o.ObjectID).Scan(&totalCount); err != nil {
		slog.Error("ChildCount", "ObjectID", o.ObjectID, "err", err.Error())
	}
	return totalCount
}

func (o *Object) Children(limit, offset int64) []*Object {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	return filteredObjects("PARENT_ID", o.ObjectID, limit, offset)
}

func (o *Object) FullPath() string {
	return mediaDir + o.Path
}

func (o *Object) Title() string {
	if o.Type == TypeFolder {
		if o.Path != "/" {
			return path.Base(o.Path)
		}
		return "root"
	}
	return fs_utils.NameWithoutExtension(path.Base(o.Path))
}

func (o *Object) MimeType() string {
	if o.Type == TypeVideo {
		v := o.Meta.(*VideoMeta)
		if strings.Contains(v.Format, "matroska") || strings.Contains(v.Format, "avi") {
			return "video/avi"
		}
		return "video/x-msvideo"
	}
	if o.Type == TypeStream {
		v := o.Meta.(*StreamMeta)
		if v.MimeType != "" {
			return v.MimeType
		}
		return "video/mp4"
	}
	return "video/mp4"
}

func (o *Object) Profile() string {
	if o.Type == TypeVideo {
		return ""
	}
	if o.Type == TypeStream {
		return o.Meta.(*StreamMeta).Profile
	}
	return ""
}

func (o *Object) SetBookmark(bm uint64) {
	if _, err := db.Exec(`UPDATE OBJECTS SET BOOKMARK = ? WHERE OBJECT_ID = ?`, bm, o.ObjectID); err != nil {
		slog.Error("set bookmark", "err", err.Error(), "obj", o.ObjectID, "bm", bm)
	}
	// remove all cached thumbnails (bookmark value used for generation thumbnail)
	removeObjectCache(o.ObjectID)
}

func filteredObjects(f, v string, limit, offset int64) []*Object {
	var rows *sql.Rows
	var err error
	items := make([]*Object, 0)
	c := strings.Join([]string{
		"OBJECT_ID", /*1*/
		"PARENT_ID", /*2*/
		"PATH",      /*3*/
		"TYPE",      /*4*/
		"TIMESTAMP", /*4*/
		"BOOKMARK",  /*5*/
		"META",      /*6*/
	}, ",")
	q := "SELECT " + c + " FROM OBJECTS WHERE " + f + " = ? ORDER BY TYPE, PATH LIMIT ? OFFSET ?"
	if rows, err = db.Query(q, v, limit, offset); err != nil {
		slog.Error("filteredObjects.select", f, v, "err", err.Error())
		return items
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			slog.Error("filteredObjects.close", f, v, "err", err.Error())
		}
	}(rows)

	for rows.Next() {
		var m []byte
		item := &Object{}
		err = rows.Scan(
			&item.ObjectID,  /*1*/
			&item.ParentID,  /*2*/
			&item.Path,      /*3*/
			&item.Type,      /*4*/
			&item.Timestamp, /*5*/
			&item.Bookmark,  /*6*/
			&m,              /*7*/
		)
		if err != nil {
			slog.Error("filteredObjects.scan", f, v, "err", err.Error())
			return items
		}
		if m != nil {
			if item.Type == TypeVideo {
				var meta VideoMeta
				if err = json.Unmarshal(m, &meta); err != nil {
					slog.Error("filteredObjects.video.unmarshal", f, v, "err", err.Error())
				} else {
					item.Meta = &meta
				}
			} else if item.Type == TypeStream {
				var meta StreamMeta
				if err = json.Unmarshal(m, &meta); err != nil {
					slog.Error("filteredObjects.stream.unmarshal", f, v, "err", err.Error())
				} else {
					item.Meta = &meta
				}
			}
		}
		items = append(items, item)
	}
	return items
}
