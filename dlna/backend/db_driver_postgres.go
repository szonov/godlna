package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDriver struct {
	db *pgxpool.Pool
}

func NewPostgresDriver(db *pgxpool.Pool) *PostgresDriver {
	return &PostgresDriver{
		db: db,
	}
}
func (d *PostgresDriver) GetObjects(f ObjectSearchFilter) (*ObjectSearchResponse, error) {
	out := &ObjectSearchResponse{
		Items: make([]*Object, 0),
	}

	idx := 1
	where := make([]string, 0)
	params := make([]any, 0)

	if f.ID > 0 {
		where = append(where, fmt.Sprintf("id = %d", f.ID))
	}

	if f.LastVisitedId > 0 {
		where = append(where, fmt.Sprintf("id > %d", f.LastVisitedId))
	}

	if f.ParentPath != "" {
		where = append(where, fmt.Sprintf("path LIKE $%d", idx))
		params = append(params, f.ParentPath+"/%")
		idx++
		where = append(where, fmt.Sprintf("path NOT LIKE $%d", idx))
		params = append(params, f.ParentPath+"/%/%")
		idx++
	}

	if f.OwnPaths != nil && len(f.OwnPaths) > 0 {
		q := make([]string, len(f.OwnPaths))
		for i, p := range f.OwnPaths {
			q[i] = fmt.Sprintf("$%d", idx)
			params = append(params, p)
			idx++
		}
		where = append(where, "path IN ("+strings.Join(q, ",")+")")
	}

	switch f.Status {
	case StatusPublic:
		where = append(where, "reindex_at IS NULL")
	case StatusDirty:
		where = append(where, "reindex_at IS NOT NULL")
	case StatusReindex:
		where = append(where, "reindex_at IS NOT NULL AND reindex_at <= now()")
	case StatusAll:
		// no restrictions
	}

	var whereString string
	if len(where) > 0 {
		whereString = fmt.Sprintf(" WHERE %s", strings.Join(where, " AND "))
	}

	if f.WithTotalMatches {
		q := "SELECT" + " count(*) FROM objects" + whereString
		if err := d.db.QueryRow(context.Background(), q, params...).Scan(&out.TotalMatches); err != nil {
			return nil, fmt.Errorf("(psql.Objects) failed getting total matches: %w", err)
		}
		if out.TotalMatches == 0 {
			return out, nil
		}
	}

	var orderBy string
	switch f.Sort {
	case SortPublic:
		orderBy = " ORDER BY typ, path"
	case SortById:
		orderBy = " ORDER BY id"
	case SortNone:
		// no sorting
	}

	q := "SELECT" + " * FROM objects" + whereString + orderBy

	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limit)
	}
	if f.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", f.Offset)
	}

	slog.Info("sql", "q", q, "params", params)

	rows, err := d.db.Query(context.Background(), q, params...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return nil, fmt.Errorf("(psql.Objects) failed query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item := new(Object)
		if err = rows.Scan(
			&item.ID,
			&item.Path,
			&item.Typ,
			&item.Format,
			&item.FileSize,
			&item.VideoCodec,
			&item.AudioCodec,
			&item.Width,
			&item.Height,
			&item.Channels,
			&item.Bitrate,
			&item.Frequency,
			&item.Duration,
			&item.Bookmark,
			&item.Date,
			&item.Online,
			&item.ReindexAt,
		); err != nil {
			return nil, fmt.Errorf("(psql.Objects) failed scan row: %w", err)
		}
		out.Items = append(out.Items, item)
	}

	return out, nil
}

func (d *PostgresDriver) UpdateObject(o *Object, v *VideoInfo, b *BookmarkInfo) error {
	if o == nil || o.ID == 0 {
		return fmt.Errorf("(psql.UpdateVideoInfo) nil object")
	}

	if v == nil && b == nil {
		// nothing to update
		return nil
	}

	idx := 1
	updates := make([]string, 0)
	params := make([]any, 0)

	if v != nil {
		// videoInfo
		if v.Format != o.Format {
			o.Format = v.Format
			updates = append(updates, fmt.Sprintf("format = $%d", idx))
			params = append(params, v.Format)
			idx++
		}
		if v.FileSize != o.FileSize {
			o.FileSize = v.FileSize
			updates = append(updates, fmt.Sprintf("file_size = %d", v.FileSize))
		}
		if v.VideoCodec != o.VideoCodec {
			o.VideoCodec = v.VideoCodec
			updates = append(updates, fmt.Sprintf("video_codec = $%d", idx))
			params = append(params, v.VideoCodec)
			idx++
		}
		if v.AudioCodec != o.AudioCodec {
			o.AudioCodec = v.AudioCodec
			updates = append(updates, fmt.Sprintf("audio_codec = $%d", idx))
			params = append(params, v.AudioCodec)
			idx++
		}
		if v.Width != o.Width {
			o.Width = v.Width
			updates = append(updates, fmt.Sprintf("width = %d", v.Width))
		}
		if v.Height != o.Height {
			o.Height = v.Height
			updates = append(updates, fmt.Sprintf("height = %d", v.Height))
		}
		if v.Channels != o.Channels {
			o.Channels = v.Channels
			updates = append(updates, fmt.Sprintf("channels = %d", v.Channels))
		}
		if v.Bitrate != o.Bitrate {
			o.Bitrate = v.Bitrate
			updates = append(updates, fmt.Sprintf("bitrate = %d", v.Bitrate))
		}
		if v.Frequency != o.Frequency {
			o.Frequency = v.Frequency
			updates = append(updates, fmt.Sprintf("frequency = %d", v.Frequency))
		}
		if v.Duration != o.Duration {
			o.Duration = v.Duration
			updates = append(updates, fmt.Sprintf("duration = %d", v.Duration))
		}
		if v.Date != o.Date {
			o.Date = v.Date
			updates = append(updates, fmt.Sprintf("date = %d", v.Date))
		}
		if o.ReindexAt.Valid {
			o.ReindexAt = sql.NullTime{}
			updates = append(updates, "reindex_at = NULL")
		}
	}
	if b != nil {
		// bookmarkInfo
		if o.Bookmark != b.Bookmark {
			o.Bookmark = b.Bookmark
			if b.Bookmark.Valid {
				updates = append(updates, fmt.Sprintf("bookmark = %d", b.Bookmark.Int64))
			} else {
				updates = append(updates, fmt.Sprintf("bookmark = NULL"))
			}
		}
	}

	if len(updates) == 0 {
		// nothing to update
		//slog.Debug("Nothing to update")
		return nil
	}

	q := fmt.Sprintf("UPDATE"+" objects SET %s WHERE id = %d", strings.Join(updates, ", "), o.ID)

	slog.Debug("UPDATE", "q", q)

	if _, err := d.db.Exec(context.Background(), q, params...); err != nil {
		return fmt.Errorf("(psql.UpdateVideoInfo) failed query: %w", err)
	}

	return nil
}

func (d *PostgresDriver) AllObjectsToOffline() error {
	_, err := d.db.Exec(context.Background(), "UPDATE objects SET online = false WHERE online")
	return err
}

func (d *PostgresDriver) DeleteOfflineObjects() error {
	_, err := d.db.Exec(context.Background(), "DELETE FROM objects WHERE NOT online")
	return err
}

func (d *PostgresDriver) Index(isDir bool, fullPath string) error {
	_, err := d.db.Exec(context.Background(), "CALL index_add($1, $2)", isDir, fullPath)
	return err
}

func (d *PostgresDriver) Remove(isDir bool, fullPath string) error {
	_, err := d.db.Exec(context.Background(), "CALL index_delete($1, $2)", isDir, fullPath)
	return err
}

func (d *PostgresDriver) Rename(isDir bool, oldFullPath string, newFullPath string) error {
	_, err := d.db.Exec(context.Background(), "CALL index_rename($1, $2, $3)", isDir, oldFullPath, newFullPath)
	return err
}
