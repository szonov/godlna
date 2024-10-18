package backend

import (
	"database/sql"
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/types"
	"log/slog"
	"os"
	"path"
	"strings"
)

const (
	Folder = 1
	Video  = 2
)

var (
	MediaDir string
	CacheDir string
	DB       *sql.DB
)

type (
	Object struct {
		ID         int64
		ObjectID   string
		ParentID   string
		Type       int
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
)

// GetSystemUpdateId is SystemUpdateID for ContentDirectory service
// Investigations shows that no one DLNA client handle this as described in documentation,
// even more... my Samsung TVs do not use eventing at all.
// So, handling update ID for containers and system update id is useless,
// Supporting real UpdateID only add not necessary complexity to logic of app...
func GetSystemUpdateId() string {
	return "10"
}

// GetObjectCacheDir returns path to directory where all object's cached resources stored
func GetObjectCacheDir(objectID string) string {
	return path.Join(CacheDir, "thumbs", strings.Replace(objectID, "$", "/", -1))
}

func (o *Object) FullPath() string {
	return MediaDir + o.Path
}

func (o *Object) BookmarkPercent() uint8 {
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
	if o.Type == Folder {
		return path.Base(o.Path)
	}
	return fs_utils.NameWithoutExtension(path.Base(o.Path))
}

func (o *Object) Children(limit, offset int64) ([]*Object, uint64) {
	var totalCount uint64
	q := `SELECT COUNT(*) FROM OBJECTS WHERE PARENT_ID = ?`
	if err := DB.QueryRow(q, o.ObjectID).Scan(&totalCount); err != nil || totalCount == 0 {
		return make([]*Object, 0), 0
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	return getFilteredObjects("PARENT_ID", o.ObjectID, limit, offset), totalCount
}

func GetObject(objectID string) *Object {
	o := getFilteredObjects("OBJECT_ID", objectID, 1, 0)
	if len(o) > 0 {
		return o[0]
	}
	return nil
}

func getFilteredObjects(f, v string, limit, offset int64) []*Object {
	var rows *sql.Rows
	var err error
	items := make([]*Object, 0)
	c := strings.Join([]string{
		"OBJECT_ID",   /*1*/
		"PARENT_ID",   /*2*/
		"TYPE",        /*3*/
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
	q := "SELECT " + c + " FROM OBJECTS WHERE " + f + " = ? ORDER BY TYPE, PATH LIMIT ? OFFSET ?"
	if rows, err = DB.Query(q, v, limit, offset); err != nil {
		slog.Error("objects::select", "err", err.Error())
		return items
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			slog.Error("objects::select.rows.close", "err", err.Error())
		}
	}(rows)

	for rows.Next() {
		item := &Object{}
		err = rows.Scan(
			&item.ObjectID,   /*1*/
			&item.ParentID,   /*2*/
			&item.Type,       /*3*/
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
			slog.Error("objects::select.rows.scan", "err", err.Error())
			return items
		}
		items = append(items, item)
	}
	return items
}

func SetBookmark(objectID string, posSecond uint64) {
	if posSecond > 60 {
		// short time slot to remember next time what I watched
		posSecond -= 8
	}

	if _, err := DB.Exec(`UPDATE OBJECTS SET BOOKMARK = ? WHERE OBJECT_ID = ?`, posSecond, objectID); err != nil {
		slog.Error("set bookmark", "err", err.Error(), "obj", objectID, "pos", posSecond)
	}

	// remove all cached thumbnails (bookmark value used for generation thumbnail)
	_ = os.RemoveAll(GetObjectCacheDir(objectID))
}
