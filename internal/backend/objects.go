package backend

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"log/slog"
	"time"
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
		ID          int64
		ObjectID    string
		ParentID    string
		Type        int
		Title       string
		Path        string
		Timestamp   *NullableNumber
		UpdateID    uint64
		Size        *NullableNumber
		Resolution  *NullableString
		Channels    *NullableNumber
		SampleRate  *NullableNumber
		BitRate     *NullableNumber
		Bookmark    *NullableNumber
		DurationSec *Duration
		MimeType    *NullableString
	}

	Duration       float64
	NullableNumber uint64
	NullableString string
)

func (o *Object) FullPath() string {
	return MediaDir + o.Path
}

func (o *Object) BookmarkPercent() uint8 {
	percent := o.Bookmark.Uint64()
	duration := o.DurationSec.Uint64()
	if percent > 0 && duration > 0 {
		return uint8(100 * percent / duration)
	}
	return 0
}

func (d *Duration) Duration() time.Duration {
	if d != nil {
		return time.Duration(float64(*d) * float64(time.Second))
	}
	return 0
}

func (d *Duration) Uint64() uint64 {
	if d != nil {
		return uint64(*d)
	}
	return 0
}

func (d *Duration) String() string {
	dur := d.Duration()
	ms := dur.Milliseconds() % 1000
	s := int(dur.Seconds()) % 60
	m := int(dur.Minutes()) % 60
	h := int(dur.Hours())

	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

func (n *NullableNumber) String() string {
	return fmt.Sprintf("%d", n.Int64())
}

func (n *NullableNumber) Uint64() uint64 {
	if n != nil {
		return uint64(*n)
	}
	return 0
}
func (n *NullableNumber) Int() int {
	if n != nil {
		return int(*n)
	}
	return 0
}
func (n *NullableNumber) Uint() uint {
	if n != nil {
		return uint(*n)
	}
	return 0
}

func (n *NullableNumber) Int64() int64 {
	if n != nil {
		return int64(*n)
	}
	return 0
}

func (n *NullableNumber) Time() time.Time {
	return time.Unix(n.Int64(), 0)
}

func (s *NullableString) String() string {
	if s != nil {
		return string(*s)
	}
	return ""
}

func GetObject(objectID string) *Object {
	objects, _ := getFilteredObjects(objectID, "", 1, 0, false)
	if len(objects) == 1 {
		return objects[0]
	}
	return nil
}

func GetObjectChildren(objectID string, limit, offset int64) ([]*Object, uint64) {
	return getFilteredObjects("", objectID, limit, offset, true)
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
		"OBJECT_ID",    /*1*/
		"PARENT_ID",    /*2*/
		"TYPE",         /*3*/
		"TITLE",        /*4*/
		"TIMESTAMP",    /*5*/
		"SIZE",         /*6*/
		"RESOLUTION",   /*7*/
		"CHANNELS",     /*8*/
		"SAMPLE_RATE",  /*9*/
		"BITRATE",      /*10*/
		"BOOKMARK",     /*11*/
		"DURATION_SEC", /*12*/
		"PATH",         /*13*/
		"MIME",         /*14*/
	).
		From("OBJECTS").
		Where(where).
		OrderBy("TYPE", "TITLE").
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
			&item.ObjectID,    /*1*/
			&item.ParentID,    /*2*/
			&item.Type,        /*3*/
			&item.Title,       /*4*/
			&item.Timestamp,   /*5*/
			&item.Size,        /*6*/
			&item.Resolution,  /*7*/
			&item.Channels,    /*8*/
			&item.SampleRate,  /*9*/
			&item.BitRate,     /*10*/
			&item.Bookmark,    /*11*/
			&item.DurationSec, /*12*/
			&item.Path,        /*13*/
			&item.MimeType,    /*14*/
		)
		if err != nil {
			slog.Error("scan error", "err", err.Error())
			return items, totalCount
		}
		items = append(items, item)
	}

	return items, totalCount
}

func GetObjectPathMime(objectID string) (string, string) {
	var path string
	var mime *NullableString
	_ = DB.QueryRow(`SELECT PATH, MIME FROM OBJECTS WHERE OBJECT_ID = ?`, objectID).Scan(&path, &mime)
	if path != "" {
		return MediaDir + path, mime.String()
	}
	return "", ""
}

func SetBookmark(objectID string, posSecond uint64) {
	_, err := sq.Update("OBJECTS").Set("BOOKMARK", posSecond).Where(sq.Eq{"OBJECT_ID": objectID}).
		RunWith(DB).Exec()

	if err != nil {
		slog.Error("set bookmark", "err", err.Error())
	}

	// todo - thumb manipulations
}
