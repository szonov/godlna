package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/szonov/godlna/internal/fs_utils"
	"gopkg.in/vansante/go-ffprobe.v2"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type scanner struct {
	oID   string
	oPath string
	oType int
	mu    sync.Mutex
}

var Scanner = new(scanner)

func (s *scanner) Scan(objectID string) {
	// only one scanner can be run simultaneously
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	s.oID = objectID

	if err = s.beforeScan(); err != nil {
		slog.Error("scanner::beforeScan", "err", err.Error())
		return
	}

	slog.Debug("scanner::start", "ID", s.oID, "path", s.oPath, "type", s.oType)

	if err = s.run(); err != nil {
		slog.Error("scanner::run", "err", err.Error())
		return
	}

	if err = s.afterScan(); err != nil {
		slog.Error("scanner::afterScan", "err", err.Error())
		return
	}

	slog.Debug("scanner::complete")
}

func (s *scanner) beforeScan() (err error) {
	var q string

	q = `UPDATE OBJECTS SET TO_DELETE = 1 WHERE OBJECT_ID LIKE ?`
	if err = execQuery(err, q, s.oID+"$%"); err != nil {
		return err
	}
	q = `SELECT TYPE, PATH FROM OBJECTS WHERE OBJECT_ID = ?`
	err = DB.QueryRow(q, s.oID).Scan(&s.oType, &s.oPath)
	return
}

func (s *scanner) run() (err error) {
	startPath := path.Join(MediaDir, s.oPath)

	var entry os.FileInfo
	entry, err = os.Stat(startPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && s.oID != "0" {
			// file or directory deleted - should delete object with children completely
			// but only if not media root object
			err = execQuery(nil, `UPDATE OBJECTS SET TO_DELETE = 1 WHERE OBJECT_ID = ?`, s.oID)
		}
		return
	}
	if entry.IsDir() {
		err = s.readDir(startPath)
	}
	return
}
func (s *scanner) afterScan() error {
	levels := s.getLevelsToDelete()
	for _, level := range levels {
		if err := s.deleteLevel(level); err != nil {
			return err
		}
	}
	return nil
}

func (s *scanner) readDir(dir string) error {

	var entries []fs.DirEntry
	var info fs.FileInfo
	var err error

	if entries, err = os.ReadDir(dir); err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := path.Join(dir, entry.Name())

		// can not get info about file/directory - just skip it
		if info, err = entry.Info(); err != nil {
			slog.Error("entry.Info", "path", fullPath, "err", err.Error())
			continue
		}

		if entry.IsDir() {
			if err = s.checkFolder(fullPath); err != nil {
				slog.Error("checkFolder", "path", fullPath, "err", err.Error())
				continue
			}
			if err = s.readDir(fullPath); err != nil {
				slog.Error("readDir", "path", fullPath, "err", err.Error())
				continue
			}
		} else if fs_utils.IsVideoFile(entry.Name()) {
			if err = s.checkVideo(fullPath, info.ModTime()); err != nil {
				slog.Error("checkVideo", "path", fullPath, "err", err.Error())
				continue
			}
		}
	}
	return nil
}

func (s *scanner) checkFolder(fullPath string) (err error) {
	relPath := strings.TrimPrefix(fullPath, MediaDir)

	slog.Debug("FOLDER", "path", relPath)

	var objectID string
	var parentID string

	if s.removeToDeleteFlag(relPath, Folder) {
		return
	}
	if objectID, parentID, err = s.generateNewObjectId(relPath); err != nil {
		return
	}

	err = insertObject(map[string]any{
		"OBJECT_ID": objectID,
		"PARENT_ID": parentID,
		"TYPE":      Folder,
		"PATH":      relPath,
	})

	return
}

func (s *scanner) checkVideo(fullPath string, modTime time.Time) (err error) {
	relPath := strings.TrimPrefix(fullPath, MediaDir)

	slog.Debug("> VIDEO", "path", relPath)

	var objectID string
	var parentID string

	if s.removeToDeleteFlag(relPath, Video) {
		return
	}
	if objectID, parentID, err = s.generateNewObjectId(relPath); err != nil {
		return
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	var ffdata *ffprobe.ProbeData
	if ffdata, err = ffprobe.ProbeURL(ctx, fullPath); err != nil {
		err = fmt.Errorf("ffprobe '%s' : %w", fullPath, err)
		return
	}

	vStream := ffdata.FirstVideoStream()
	aStream := ffdata.FirstAudioStream()

	if vStream == nil || aStream == nil {
		err = fmt.Errorf("video or audio steam is empty '%s'", fullPath)
		return
	}

	size, _ := strconv.ParseUint(ffdata.Format.Size, 10, 64)
	sampleRate, _ := strconv.ParseUint(aStream.SampleRate, 10, 64)
	var bitrate uint64
	if bitrate, err = strconv.ParseUint(ffdata.Format.BitRate, 10, 64); err == nil {
		if bitrate > 8 {
			bitrate = bitrate / 8
		}
	}

	err = insertObject(map[string]any{
		"OBJECT_ID":   objectID,
		"PARENT_ID":   parentID,
		"TYPE":        Video,
		"PATH":        relPath,
		"TIMESTAMP":   modTime.Unix(),
		"DURATION":    uint64(ffdata.Format.DurationSeconds),
		"SIZE":        size,
		"RESOLUTION":  fmt.Sprintf("%dx%d", vStream.Width, vStream.Height),
		"CHANNELS":    aStream.Channels,
		"SAMPLE_RATE": sampleRate,
		"BITRATE":     bitrate,
		"FORMAT":      ffdata.Format.FormatName,
		"VIDEO_CODEC": vStream.CodecName,
		"AUDIO_CODEC": aStream.CodecName,
	})

	return
}

func (s *scanner) generateNewObjectId(relPath string) (objectID, parentID string, err error) {
	query := `SELECT OBJECT_ID FROM OBJECTS WHERE PATH = ? AND TYPE = ?`
	err = DB.QueryRow(query, filepath.Dir(relPath), Folder).Scan(&parentID)
	if err == nil {
		var seq int64
		err = DB.QueryRow(`SELECT seq FROM SQLITE_SEQUENCE WHERE name = 'OBJECTS'`).Scan(&seq)
		if err == nil {
			return parentID + "$" + strconv.FormatInt(seq+1, 10), parentID, nil
		}
	}
	return "", "", err
}

func (s *scanner) getLevelsToDelete() []int {
	levels := make([]int, 0)

	rows, err := DB.Query(`SELECT DISTINCT LEVEL FROM OBJECTS WHERE TO_DELETE = 1 ORDER BY LEVEL`)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("scanner::getLevelsToDelete::select.Level", "err", err.Error())
		}
		return levels
	}

	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var level int
		if err = rows.Scan(&level); err != nil {
			slog.Error("scanner::getLevelsToDelete::rows.Scan", "err", err.Error())
			return levels
		}
		levels = append(levels, level)
	}

	return levels
}

func (s *scanner) deleteLevel(level int) (err error) {
	var rows *sql.Rows
	q := `SELECT OBJECT_ID, PARENT_ID FROM OBJECTS WHERE TO_DELETE = 1 AND LEVEL = ? ORDER BY ID`
	rows, err = DB.Query(q, level)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		return
	}

	m := make([][2]string, 0)
	for rows.Next() {
		var oID, pID string
		if err = rows.Scan(&oID, &pID); err != nil {
			_ = rows.Close()
			err = fmt.Errorf("deleteLevel1 level=%d : %w", level, err)
			return
		}
		m = append(m, [2]string{oID, pID})
	}
	_ = rows.Close()

	for _, d := range m {
		if err = s.deleteObject(d[0], d[1]); err != nil {
			err = fmt.Errorf("deleteLevel3 level=%d '%s', '%s' : %w", level, d[0], d[1], err)
			break
		}
	}

	return
}

func (s *scanner) deleteObject(objectID string, parentID string) (err error) {
	_ = os.RemoveAll(GetObjectCacheDir(objectID))
	err = execQuery(nil, `DELETE FROM OBJECTS WHERE OBJECT_ID = ?`, objectID)
	return execQuery(err, `UPDATE OBJECTS SET SIZE = SIZE - 1 WHERE OBJECT_ID = ?`, parentID)
}

func (s *scanner) removeToDeleteFlag(relPath string, oType int) (removed bool) {
	q := `UPDATE OBJECTS SET TO_DELETE = 0 WHERE PATH = ? AND TYPE = ?`
	affectedCount, _ := execQueryRowsAffected(nil, q, relPath, oType)
	removed = affectedCount > 0
	return
}

//func GetParentID(objectID string) string {
//	i := strings.LastIndex(objectID, "$")
//	if i > -1 {
//		return objectID[:i]
//	}
//	if objectID == "0" {
//		return "-1"
//	}
//	return "0"
//}
