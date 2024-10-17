package backend

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/types"
	"log/slog"
	"path"
	"strconv"
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
		UpdateID   UpdateIdNumber
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

	UpdateIdNumber uint64
)

func (u UpdateIdNumber) Uint64() uint64 {
	return uint64(u)
}

func (u UpdateIdNumber) String() string {
	return strconv.FormatUint(uint64(u), 10)
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
		return "video/x-matroska"
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
	return getFilteredObjects("", o.ObjectID, limit, offset, true)
}

func GetObject(objectID string) *Object {
	objects, _ := getFilteredObjects(objectID, "", 1, 0, false)
	if len(objects) == 1 {
		return objects[0]
	}
	return nil
}

func getFilteredObjects(oid, pid string, limit, offset int64, withTotal bool) ([]*Object, uint64) {

	var err error
	var rows *sql.Rows
	var where sq.Eq

	var totalCount uint64
	items := make([]*Object, 0)

	if oid != "" {
		// find exact object
		where = sq.Eq{"OBJECT_ID": oid}
	} else if pid != "" {
		// find children
		where = sq.Eq{"PARENT_ID": pid}
	} else {
		// empty search result
		return items, totalCount
	}

	if withTotal {
		err = sq.Select("COUNT(*)").From("OBJECTS").Where(where).RunWith(DB).Scan(&totalCount)
		if err != nil || totalCount == 0 {
			return items, totalCount
		}
	}

	if offset < 0 {
		offset = 0
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err = sq.Select(
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
		"UPDATE_ID",   /*16*/
	).
		From("OBJECTS").
		Where(where).
		OrderBy("TYPE", "PATH").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		RunWith(DB).Query()

	if err != nil {
		slog.Error("select OBJECTS", "err", err.Error())
		return items, totalCount
	}

	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			slog.Debug("rows close error", "err", err.Error())
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
			&item.UpdateID,   /*16*/
		)
		if err != nil {
			slog.Error("scan error", "err", err.Error())
			return items, totalCount
		}
		items = append(items, item)
	}

	return items, totalCount
}

func SetBookmark(objectID string, posSecond uint64) {
	if posSecond > 60 {
		// short time slot to remember next time what I watched
		posSecond -= 8
	}
	slog.Debug("set bookmark", "obj", objectID, "pos", posSecond)

	var query string
	newUpdateID := GetSystemUpdateId() + 1

	// set bookmark
	query = `UPDATE OBJECTS SET BOOKMARK = ?, UPDATE_ID = ? WHERE OBJECT_ID = ?`
	if _, err := DB.Exec(query, posSecond, newUpdateID, objectID); err != nil {
		slog.Error("set bookmark", "err", err.Error(), "obj", objectID, "pos", posSecond)
	}
	// update parents' UPDATE_ID
	touchParent(objectID, newUpdateID)
	// set system's setting UPDATE_ID
	setUpdateID(newUpdateID)
	// remove thumbnails, should be generated new one
	removeThumbnails(objectID)
}

func touchParent(objectID string, newUpdateID UpdateIdNumber) {
	parentID := GetParentID(objectID)
	query := `UPDATE OBJECTS SET UPDATE_ID = ? WHERE OBJECT_ID = ?`
	if _, err := DB.Exec(query, newUpdateID, parentID); err != nil {
		slog.Error("update parent", "err", err.Error(), "objectID", objectID, "parentID", parentID)
	}
}
