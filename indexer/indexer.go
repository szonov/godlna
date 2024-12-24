package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sync"
)

const (
	TypeFolder int = 1
	TypeVideo  int = 2
)

type Indexer struct {
	dir string
	db  *pgxpool.Pool
	mu  *sync.Mutex
}

func NewIndexer(dir string, db *pgxpool.Pool) *Indexer {
	return &Indexer{
		dir: dir,
		db:  db,
		mu:  &sync.Mutex{},
	}
}

func (idx *Indexer) FullScan() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	var err error
	if _, err = idx.db.Exec(context.Background(), "UPDATE objects SET online = false"); err != nil {
		slog.Error(err.Error())
		return
	}
	if !IsPathIgnored(idx.dir) {
		if info, err := os.Stat(idx.dir); err == nil {
			idx.createOrUpdateObject(idx.dir, info)
			idx.scanDir(idx.dir)
		} else {
			_, _ = idx.db.Exec(context.Background(), "DELETE FROM objects WHERE path = $1", idx.dir)
		}
	}
	if _, err = idx.db.Exec(context.Background(), "DELETE FROM objects WHERE online = false"); err != nil {
		slog.Error(err.Error())
	}
}

func (idx *Indexer) createOrUpdateObject(fullPath string, info fs.FileInfo) {
	if info.IsDir() {
		slog.Info("-- DIR", "path", fullPath)

		query := `INSERT INTO objects (path, typ, date, online) 
		VALUES ($1, $2, $3, true) ON CONFLICT(path) DO 
		UPDATE SET typ = EXCLUDED.typ, date = EXCLUDED.date, online = EXCLUDED.online`

		_, err := idx.db.Exec(
			context.Background(),
			query,
			fullPath,
			TypeFolder,
			info.ModTime().Unix(),
		)

		if err != nil {
			slog.Error(err.Error())
		}

		return
	}

	if IsVideoFile(fullPath) {
		slog.Info("-- VID", "path", fullPath)

		var vID int64
		var vTyp int
		query := "UPDATE objects SET online = true WHERE path = $1 RETURNING id, typ"
		_ = idx.db.QueryRow(context.Background(), query, fullPath).Scan(&vID, &vTyp)
		if vID > 0 && vTyp == TypeVideo && FileExists(ThumbPath(fullPath)) {
			return
		}

		//var vID, vSize, vDate int64
		//var vTyp int
		//query := "SELECT id, typ, file_size, date FROM objects WHERE path = $1"
		//_ = db.Engine.QueryRow(context.Background(), query, fullPath).Scan(&vID, &vTyp, &vSize, &vDate)
		//if vID > 0 && vTyp == TypeVideo && vSize == info.Size() && vDate == info.ModTime().Unix() {
		//	_, err := db.Engine.Exec(context.Background(), "UPDATE objects SET online = true WHERE id = $1", vID)
		//	if err != nil {
		//		slog.Error(err.Error())
		//	}
		//	return
		//}

		ffData, err := Probe(fullPath)
		if err != nil {
			err = fmt.Errorf("ffprobe '%s' : %w", fullPath, err)
			slog.Error(err.Error())
			return
		}

		vStream := ffData.FirstVideoStream()
		aStream := ffData.FirstAudioStream()

		if vStream == nil || aStream == nil {
			err = fmt.Errorf("video or audio steam is empty '%s'", fullPath)
			slog.Error(err.Error())
			return
		}

		query = `INSERT INTO objects 
    		(path, typ, format, file_size, video_codec, audio_codec, width, height, channels, bitrate, frequency, duration, date, online) 
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, true) 
		ON CONFLICT(path) DO UPDATE SET 
			typ = EXCLUDED.typ, 
			format = EXCLUDED.format, 
			file_size = EXCLUDED.file_size, 
			video_codec = EXCLUDED.video_codec, 
			audio_codec = EXCLUDED.audio_codec, 
			width = EXCLUDED.width, 
			height = EXCLUDED.height, 
			channels = EXCLUDED.channels, 
			bitrate = EXCLUDED.bitrate, 
			frequency = EXCLUDED.frequency, 
			duration = EXCLUDED.duration, 
			date = EXCLUDED.date, 
			online = EXCLUDED.online
		RETURNING duration, bookmark`

		var duration int64
		var bookmark sql.NullInt64

		err = idx.db.QueryRow(
			context.Background(),
			query,
			fullPath,
			TypeVideo,
			ffData.Format.FormatName,
			info.Size(),
			vStream.CodecName,
			aStream.CodecName,
			vStream.Width,
			vStream.Height,
			aStream.Channels,
			ffData.Format.BitRate(),
			aStream.SampleRate,
			ffData.Format.Duration().Milliseconds(),
			info.ModTime().Unix(),
		).Scan(&duration, &bookmark)

		if err != nil {
			slog.Error(err.Error())
		} else {
			MakeDSMStyleThumbnail(fullPath, duration, bookmark, false)
		}
	}
}

func (idx *Indexer) scanDir(dir string) {

	var entries []fs.DirEntry
	var err error
	var info fs.FileInfo

	if entries, err = os.ReadDir(dir); err != nil {
		slog.Error("scanDir", "err", err)
		return
	}

	for _, entry := range entries {
		fullPath := path.Join(dir, entry.Name())
		if !IsPathIgnored(fullPath) {
			if info, err = entry.Info(); err == nil {
				idx.createOrUpdateObject(fullPath, info)
				if entry.IsDir() {
					idx.scanDir(fullPath)
				}
			}
		}
	}
	return
}
