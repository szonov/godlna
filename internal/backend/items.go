package backend

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"log/slog"
	"time"
)

type (
	Object struct {
		ID            int64
		ObjectID      string
		ParentID      string
		Class         string
		Title         string
		Path          string
		Timestamp     int64
		MetaData      string
		UpdateID      uint64
		ChildrenCount uint64
		Bookmark      uint64
	}

	ObjectFilter struct {
		ObjectID string
		ParentID string
		Limit    int64
		Offset   int64
	}
)

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

	rows, err = builder.Columns("OBJECT_ID", "PARENT_ID", "CLASS", "TITLE", "TIMESTAMP", "META_DATA", "CHILDREN_COUNT", "BOOKMARK").
		OrderBy("CLASS", "TITLE").Limit(limit).Offset(offset).
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
		item := &Object{
			Timestamp: time.Now().Unix(),
		}
		var tms *int64
		var meta *string
		err = rows.Scan(&item.ObjectID, &item.ParentID, &item.Class, &item.Title, &tms, &meta, &item.ChildrenCount, &item.Bookmark)
		if tms != nil {
			item.Timestamp = *tms
		}
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
	_ = sq.Select("PATH").From("OBJECTS").Where(sq.Eq{"OBJECT_ID": objectID}).
		RunWith(DB).QueryRow().Scan(&path)
	if path != nil {
		return *path
	}
	return ""
}

func SetBookmark(objectID string, posSecond uint64) {
	_, err := sq.Update("OBJECTS").Set("BOOKMARK", posSecond).Where(sq.Eq{"OBJECT_ID": objectID}).
		RunWith(DB).Exec()

	if err != nil {
		slog.Error("set bookmark", "err", err.Error())
	}
}
