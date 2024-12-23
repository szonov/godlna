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
	"time"
)

const (
	TypeFolder int = 1
	TypeVideo  int = 2
)

type Indexer struct {
	dir       string
	db        *pgxpool.Pool
	mu        *sync.Mutex
	chunkSize int
	watcher   *Watcher
	done      chan struct{}
}

func NewIndexer(dir string, db *pgxpool.Pool) *Indexer {
	idx := &Indexer{
		dir:       dir,
		db:        db,
		mu:        &sync.Mutex{},
		chunkSize: 100,
	}
	idx.watcher = NewWatcher(dir, idx.addToQueue)
	return idx
}

func (idx *Indexer) Start() {
	go idx.StartAndListen()
}

func (idx *Indexer) StartAndListen() {
	idx.done = make(chan struct{})
	idx.watcher.Start()
	idx.fullScan()
	idx.listenQueue()
}

func (idx *Indexer) Stop() {
	idx.watcher.Stop()
	if idx.done == nil {
		return
	}
	close(idx.done)
}

func (idx *Indexer) listenQueue() {
	idx.checkQueueAndProcess()
	for {
		timer := time.NewTimer(30 * time.Second)
		select {
		case <-idx.done:
			idx.done = nil
			return
		case <-timer.C:
		}
		idx.checkQueueAndProcess()
	}
}

func (idx *Indexer) checkQueueAndProcess() {
	slog.Info("checkQueueAndProcess")
	if idx.parseQueueChunk() > 0 {
		idx.checkQueueAndProcess()
	}
}

func (idx *Indexer) fullScan() {
	var err error
	if _, err = idx.db.Exec(context.Background(), "UPDATE objects SET online = false"); err != nil {
		slog.Error(err.Error())
		return
	}
	if !IsPathIgnored(idx.dir) {
		idx.makeIndex(idx.dir)
		idx.scanDir(idx.dir)
	}
	if _, err = idx.db.Exec(context.Background(), "DELETE FROM objects WHERE online = false"); err != nil {
		slog.Error(err.Error())
	}
}

func (idx *Indexer) createOrUpdateObject(fullPath string, info fs.FileInfo) {
	if info.IsDir() {
		slog.Info("-- DIR", "path", fullPath)

		query := `INSERT INTO objects (path, typ, date, online) 
		VALUES ($1, $2, $3, true) 
		ON CONFLICT(path) DO UPDATE SET typ = EXCLUDED.typ, date = EXCLUDED.date, online = EXCLUDED.online`

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

		query := `INSERT INTO objects 
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

func (idx *Indexer) makeIndex(fullPath string) {

	slog.Info("[makeIndex]", "path", fullPath)

	info, err := os.Stat(fullPath)
	if err != nil {
		// file or directory does not exist or inaccessible
		_, _ = idx.db.Exec(context.Background(), "DELETE FROM objects WHERE path = $1", fullPath)
		return
	}

	idx.createOrUpdateObject(fullPath, info)
}

func (idx *Indexer) addToQueue(p string) {
	if IsPathIgnored(p) {
		return
	}
	idx.mu.Lock()
	slog.Info("queue", "p", p)
	_, _ = idx.db.Exec(context.Background(), "INSERT INTO queue (path) values ($1) ON CONFLICT DO NOTHING", p)
	idx.mu.Unlock()
}

// parseQueueChunk get configured amount of queued paths and index every path
// returns amount of processed paths
func (idx *Indexer) parseQueueChunk() int {
	paths := make([]string, 0)

	idx.mu.Lock()
	query := "DELETE FROM queue WHERE id IN (SELECT id FROM queue ORDER BY id LIMIT $1) RETURNING path"
	rows, err := idx.db.Query(context.Background(), query, idx.chunkSize)

	if err != nil {
		idx.mu.Unlock()
		slog.Error(err.Error())
		return 0
	}

	for rows.Next() {
		var p string
		if err = rows.Scan(&p); err != nil {
			slog.Error("unable to scan queue row", err.Error())
		} else {
			paths = append(paths, p)
		}
	}

	rows.Close()
	idx.mu.Unlock()

	for _, fullPath := range paths {
		idx.makeIndex(fullPath)
	}

	return len(paths)
}
