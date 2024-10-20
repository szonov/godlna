package store

import (
	"database/sql"
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/types"
	"log/slog"
	"os"
	"path"
	"strings"
)

type Object struct {
	ID         int64
	ObjectID   string
	ParentID   string
	Class      string
	Path       string
	Timestamp  *types.NullableNumber
	Size       *types.NullableNumber
	Resolution *types.NullableString
	Channels   *types.NullableNumber
	SampleRate *types.NullableNumber
	BitRate    *types.NullableNumber
	Bookmark   *types.Duration
	Duration   *types.Duration
	Format     *types.NullableString
	VideoCodec *types.NullableString
	AudioCodec *types.NullableString
}

// getObjectCacheDir returns path to directory where all object's cached resources stored
func getObjectCacheDir(objectID string) string {
	return path.Join(cacheDir, "thumbs", strings.Replace(objectID, "$", "/", -1))
}

func removeObjectCache(objectID string) {
	_ = os.RemoveAll(getObjectCacheDir(objectID))
}

func GetObject(objectID string, upToDate ...bool) *Object {
	if len(upToDate) > 0 && upToDate[0] {
		directoryScanner.EnsureObjectIsUpToDate(objectID)
	}
	o := filteredObjects("OBJECT_ID", objectID, 1, 0)
	if len(o) > 0 {
		return o[0]
	}
	return nil
}

func (o *Object) IsFolder() bool {
	return o.Class == "container.storageFolder"
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

func (o *Object) ViewPercentage() uint8 {
	bm := o.Bookmark.Uint64()
	duration := o.Duration.Uint64()
	if bm > 0 && duration > 0 {
		return uint8(100 * bm / duration)
	}
	return 0
}

func (o *Object) MimeType() string {
	format := o.Format.String()

	if strings.Contains(format, "matroska") {
		return "video/avi"
		//return "video/x-matroska"
	}

	if strings.Contains(format, "avi") {
		return "video/avi"
	}

	return "video/x-msvideo"
}

func (o *Object) Title() string {
	if o.Path == "/" {
		return "root"
	}
	if o.Class == "container.storageFolder" {
		return path.Base(o.Path)
	}
	return fs_utils.NameWithoutExtension(path.Base(o.Path))
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
		"OBJECT_ID",   /*1*/
		"PARENT_ID",   /*2*/
		"CLASS",       /*3*/
		"TIMESTAMP",   /*4*/
		"SIZE",        /*5*/
		"RESOLUTION",  /*6*/
		"CHANNELS",    /*7*/
		"SAMPLE_RATE", /*8*/
		"BITRATE",     /*9*/
		"BOOKMARK",    /*10*/
		"DURATION",    /*11*/
		"PATH",        /*12*/
		"FORMAT",      /*13*/
		"VIDEO_CODEC", /*14*/
		"AUDIO_CODEC", /*15*/
	}, ",")
	q := "SELECT " + c + " FROM OBJECTS WHERE " + f + " = ? ORDER BY CLASS, PATH LIMIT ? OFFSET ?"
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
		item := &Object{}
		err = rows.Scan(
			&item.ObjectID,   /*1*/
			&item.ParentID,   /*2*/
			&item.Class,      /*3*/
			&item.Timestamp,  /*4*/
			&item.Size,       /*5*/
			&item.Resolution, /*6*/
			&item.Channels,   /*7*/
			&item.SampleRate, /*8*/
			&item.BitRate,    /*9*/
			&item.Bookmark,   /*10*/
			&item.Duration,   /*11*/
			&item.Path,       /*12*/
			&item.Format,     /*13*/
			&item.VideoCodec, /*14*/
			&item.AudioCodec, /*15*/
		)
		if err != nil {
			slog.Error("filteredObjects.scan", f, v, "err", err.Error())
			return items
		}
		items = append(items, item)
	}
	return items
}
