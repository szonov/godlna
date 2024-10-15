package backend

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"log/slog"
)

type ()

func GetObjects(filter ObjectFilter) ([]*Object, uint64) {
	var err error
	var rows *sql.Rows
	items := make([]*Object, 0)

	var totalCount uint64
	var limit uint64
	var offset uint64

	if filter.ObjectID == "" && filter.ParentID == "" {
		return items, totalCount
	}

	builder := sq.Select().From("OBJECTS")

	if filter.ObjectID != "" {
		builder = builder.Where(sq.Eq{"OBJECT_ID": filter.ObjectID})
	}

	if filter.ParentID != "" {
		builder = builder.Where(sq.Eq{"PARENT_ID": filter.ParentID})
	}

	row := builder.Columns("COUNT(*)").RunWith(DB).QueryRow()
	if err = row.Scan(&totalCount); err != nil {
		slog.Error("select total", "err", err.Error())
		return items, totalCount
	}

	if totalCount == 0 {
		return items, totalCount
	}

	if filter.Offset < 0 {
		offset = 0
	} else {
		offset = uint64(filter.Offset)
	}

	if filter.Limit <= 0 {
		limit = 10
	} else if filter.Limit > 200 {
		limit = 200
	} else {
		limit = uint64(filter.Limit)
	}

	rows, err = builder.Columns(
		"OBJECT_ID",    /*1*/
		"PARENT_ID",    /*2*/
		"TYPE",         /*3*/
		"TITLE",        /*4*/
		"TIMESTAMP",    /*5*/
		"META_DATA",    /*6*/
		"SIZE",         /*7*/
		"RESOLUTION",   /*8*/
		"CHANNELS",     /*9*/
		"SAMPLE_RATE",  /*10*/
		"BITRATE",      /*11*/
		"BOOKMARK",     /*12*/
		"DURATION_SEC", /*13*/
		"PATH",         /*14*/
		"MIME",         /*15*/
	).
		OrderBy("TYPE", "TITLE").Limit(limit).Offset(offset).
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
		var meta *string
		err = rows.Scan(
			&item.ObjectID,    /*1*/
			&item.ParentID,    /*2*/
			&item.Type,        /*3*/
			&item.Title,       /*4*/
			&item.Timestamp,   /*5*/
			&meta,             /*6*/
			&item.Size,        /*7*/
			&item.Resolution,  /*8*/
			&item.Channels,    /*9*/
			&item.SampleRate,  /*10*/
			&item.BitRate,     /*11*/
			&item.Bookmark,    /*12*/
			&item.DurationSec, /*13*/
			&item.Path,        /*14*/
			&item.MimeType,    /*15*/
		)
		if meta != nil {
			item.MetaData = *meta
		}

		if err != nil {
			slog.Error("scan error", "err", err.Error())
			return items, 0
		}
		items = append(items, item)
	}

	return items, totalCount
}

func GetObjectPath(objectID string) string {
	var path *string
	_ = DB.QueryRow(`SELECT PATH FROM OBJECTS WHERE OBJECT_ID = ?`, objectID).Scan(&path)
	if path != nil {
		return MediaDir + *path
	}
	return ""
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

func GetObjectPathPercentSeen(objectID string) (string, uint8, bool) {
	var path *string
	var bookmark uint64
	var dur *Duration

	_ = DB.QueryRow(`SELECT PATH, BOOKMARK, DURATION_SEC FROM OBJECTS WHERE OBJECT_ID = ?`, objectID).
		Scan(&path, &bookmark, &dur)
	if path != nil {
		var percent uint64 = 0
		if bookmark > 0 {
			percent = 100 * bookmark / dur.Uint64()
		}
		return MediaDir + *path, uint8(percent), false
	}
	return "", 0, false
}

func SetBookmark(objectID string, posSecond uint64) {
	_, err := sq.Update("OBJECTS").Set("BOOKMARK", posSecond).Where(sq.Eq{"OBJECT_ID": objectID}).
		RunWith(DB).Exec()

	if err != nil {
		slog.Error("set bookmark", "err", err.Error())
	}

	// todo - thumb manipulations
}
