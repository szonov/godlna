package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/szonov/godlna/internal/ffmpeg"
	"github.com/szonov/godlna/internal/types"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var cacheLifeTime time.Duration
var scanGuard *Guard

var videoExtensions = []string{
	".mpg", ".mpeg", ".avi", ".mkv", ".mp4", ".m4v",
	".divx", ".asf", ".wmv", ".mts", ".m2ts", ".m2t",
	".vob", ".ts", ".flv", ".xvid", ".mov", ".3gp", ".rm", ".rmvb", ".webm",
}

type scanCachedObject struct {
	oid  string
	hash string
}

type scanFileInfo struct {
	typ  int
	skip bool
	size int64
	tm   time.Time
	hash string
	err  error
}

func isVideoFile(file string) bool {
	fileExt := strings.ToLower(filepath.Ext(file))
	for _, ext := range videoExtensions {
		if fileExt == ext {
			return true
		}
	}
	return false
}

//func isLiveStreamFile(file string) bool {
//	return strings.ToLower(filepath.Ext(file)) == ".json"
//}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func makeHash(typ int, modTime int64, size int64) string {
	return strings.Join([]string{intStr(int64(typ)), intStr(modTime), intStr(size)}, ":")
}

func getDirEntryInfo(dir string, entry fs.DirEntry) scanFileInfo {
	if entry.IsDir() {
		// synology special folders
		if entry.Name() == "@eaDir" {
			return scanFileInfo{skip: true}
		}

		if strings.HasSuffix(entry.Name(), ".stream") {
			jsonPath := path.Join(dir, entry.Name(), "stream.json")
			if s, err := os.Stat(jsonPath); err == nil {
				return scanFileInfo{
					typ:  TypeStream,
					size: s.Size(),
					tm:   s.ModTime(),
					hash: makeHash(TypeStream, s.ModTime().Unix(), s.Size()),
				}
			}
		}

		return scanFileInfo{
			typ:  TypeFolder,
			hash: makeHash(TypeFolder, 0, 0),
		}
	}

	info, err := entry.Info()
	if err != nil {
		return scanFileInfo{skip: true, err: err}
	}

	if isVideoFile(entry.Name()) {
		return scanFileInfo{
			typ:  TypeVideo,
			size: info.Size(),
			tm:   info.ModTime(),
			hash: makeHash(TypeVideo, info.ModTime().Unix(), info.Size()),
		}
	}
	//
	//if isLiveStreamFile(entry.Name()) {
	//	return scanFileInfo{
	//		typ:  TypeStream,
	//		size: info.Size(),
	//		tm:   info.ModTime(),
	//		hash: makeHash(TypeStream, info.ModTime().Unix(), info.Size()),
	//	}
	//}

	return scanFileInfo{skip: true}
}

func EnsureObjectIsUpToDate(objectID string) {
	if scanGuard.TryLock(objectID) {
		if isOutdated(objectID) {
			rescan(objectID)
		}
		scanGuard.Unlock(objectID)
	} else {
		slog.Warn("CAN NOT GET LOCK", "OBJECT_ID", objectID)
	}
}

func isOutdated(objectID string) bool {
	var tm *types.NullableNumber
	var typ int

	err := db.QueryRow("SELECT TYPE, TIMESTAMP FROM OBJECTS WHERE OBJECT_ID = ?", objectID).Scan(&typ, &tm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && objectID == "0" {
			createRootObject()
			return true
		}
		slog.Error("isOutdated", "OBJECT_ID", objectID, "err", err.Error())
		return false
	}
	if typ == TypeFolder {
		if tm == nil || tm.Time().Add(cacheLifeTime).Before(time.Now()) {
			return true
		}
	}
	return false
}

func createRootObject() {
	err := insert("OBJECTS", map[string]any{
		"OBJECT_ID": "0",
		"PARENT_ID": "-1",
		"PATH":      "/",
		"TYPE":      TypeFolder,
	})
	if err != nil {
		slog.Error("createRootObject", "err", err.Error())
	}
}

func insert(table string, data map[string]any) error {
	l := len(data)
	if l == 0 {
		return fmt.Errorf("no data to insert")
	}
	c := make([]string, l)
	m := make([]string, l)
	args := make([]any, l)
	i := 0
	for k, v := range data {
		c[i] = k
		m[i] = "?"
		args[i] = v
		i++
	}
	q := "INSERT INTO" + " " + table + " (" + strings.Join(c, ",") + ") VALUES (" + strings.Join(m, ",") + ")"
	if _, err := db.Exec(q, args...); err != nil {
		return err
	}
	return nil
}

func cachedChildren(objectID string) map[string]scanCachedObject {
	m := make(map[string]scanCachedObject)
	q := "SELECT OBJECT_ID, PATH, TYPE, TIMESTAMP, SIZE FROM OBJECTS WHERE PARENT_ID = ?"

	rows, err := db.Query(q, objectID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("cachedChildren", "OBJECT_ID", objectID, "error", err.Error())
		}
		return m
	}

	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			slog.Error("cachedChildren.close", "OBJECT_ID", objectID, "err", err.Error())
		}
	}(rows)

	for rows.Next() {
		var p, oid, hash string
		var typ int
		var size int64
		var tm *types.NullableNumber
		if err = rows.Scan(&oid, &p, &typ, &tm, &size); err != nil {
			slog.Error("cachedChildren.scan", "OBJECT_ID", objectID, "err", err.Error())
			return m
		}
		if typ == TypeFolder {
			hash = makeHash(typ, 0, 0)
		} else {
			hash = makeHash(typ, tm.Time().Unix(), size)
		}
		m[path.Join(mediaDir, p)] = scanCachedObject{oid: oid, hash: hash}
	}
	return m
}

func deleteOldChildren(objects map[string]scanCachedObject) {
	for _, o := range objects {
		deleteObject(o.oid)
	}
}

func deleteObject(objectID string) {
	if objectID != "0" {
		_, err := db.Exec("DELETE FROM OBJECTS WHERE OBJECT_ID = ? OR OBJECT_ID LIKE ?", objectID, objectID+"$%")
		if err != nil {
			slog.Error("deleteObject", "OBJECT_ID", objectID, "error", err.Error())
			return
		}
		removeObjectCache(objectID)
	}
}

func setJustUpdated(objectID string) {
	q := `UPDATE OBJECTS SET TIMESTAMP = ?  WHERE OBJECT_ID = ?`
	if _, err := db.Exec(q, time.Now().Unix(), objectID); err != nil {
		slog.Error("setJustUpdated", "OBJECT_ID", objectID, "err", err.Error())
	}
}

func generateNewObjectId(relPath string) (objectID, parentID string, err error) {
	query := `SELECT OBJECT_ID FROM OBJECTS WHERE PATH = ? AND TYPE = ?`
	err = db.QueryRow(query, filepath.Dir(relPath), TypeFolder).Scan(&parentID)
	if err == nil {
		var seq int64
		err = db.QueryRow(`SELECT seq FROM SQLITE_SEQUENCE WHERE name = 'OBJECTS'`).Scan(&seq)
		if err == nil {
			return parentID + "$" + intStr(seq+1), parentID, nil
		}
	}
	return "", "", err
}

func rescan(objectID string) {
	var dir string
	var err error
	if err = db.QueryRow("SELECT PATH FROM OBJECTS WHERE OBJECT_ID = ?", objectID).Scan(&dir); err != nil {
		slog.Error("scanDir", "OBJECT_ID", objectID, "err", err.Error())
		return
	}
	dir = path.Join(mediaDir, dir)

	slog.Debug("scanDir", "OBJECT_ID", objectID, "dir", dir)

	oldChildren := cachedChildren(objectID)

	var entries []fs.DirEntry

	if entries, err = os.ReadDir(dir); err != nil {
		deleteObject(objectID)
		deleteOldChildren(oldChildren)
		return
	}

	for _, entry := range entries {
		fullPath := path.Join(dir, entry.Name())
		info := getDirEntryInfo(dir, entry)
		if info.skip {
			if info.err != nil {
				slog.Error("entry.Info", "path", fullPath, "err", info.err.Error())
			}
			continue
		}
		if oldObject, exists := oldChildren[fullPath]; exists {
			if oldObject.hash == info.hash {
				// file or dir is not changed - skip processing
				delete(oldChildren, fullPath)
				continue
			}
		}

		switch info.typ {
		case TypeFolder:
			err = addFolder(fullPath, info)
		case TypeVideo:
			err = addVideo(fullPath, info)
		case TypeStream:
			err = addStream(fullPath, info)
		}
		if err != nil {
			slog.Error("addItem", "path", fullPath, "err", err.Error())
		}
	}

	deleteOldChildren(oldChildren)
	setJustUpdated(objectID)
}

func addFolder(fullPath string, info scanFileInfo) (err error) {
	relPath := strings.TrimPrefix(fullPath, mediaDir)
	slog.Debug("FOLDER", "path", relPath)

	var objectID string
	var parentID string

	if objectID, parentID, err = generateNewObjectId(relPath); err != nil {
		return
	}

	err = insert("OBJECTS", map[string]any{
		"OBJECT_ID": objectID,
		"PARENT_ID": parentID,
		"PATH":      relPath,
		"TYPE":      info.typ,
	})

	return
}

func addVideo(fullPath string, info scanFileInfo) (err error) {
	relPath := strings.TrimPrefix(fullPath, mediaDir)
	slog.Debug("> VIDEO", "path", relPath)

	var objectID string
	var parentID string

	if objectID, parentID, err = generateNewObjectId(relPath); err != nil {
		return
	}

	var ffData *ffmpeg.ProbeData
	ffData, err = ffmpeg.Probe(fullPath)
	if err != nil {
		err = fmt.Errorf("ffprobe '%s' : %w", fullPath, err)
		return
	}

	vStream := ffData.FirstVideoStream()
	aStream := ffData.FirstAudioStream()

	if vStream == nil || aStream == nil {
		err = fmt.Errorf("video or audio steam is empty '%s'", fullPath)
		return
	}

	err = insert("OBJECTS", map[string]any{
		"OBJECT_ID": objectID,
		"PARENT_ID": parentID,
		"PATH":      relPath,
		"TYPE":      TypeVideo,
		"TIMESTAMP": info.tm.Unix(),
		"SIZE":      info.size,
		"META": &VideoMeta{
			Resolution: vStream.Resolution(),
			Channels:   aStream.Channels,
			SampleRate: aStream.SampleRate,
			BitRate:    ffData.Format.BitRate(),
			Duration:   types.NewDuration(ffData.Format.Duration().Milliseconds()),
			Format:     ffData.Format.FormatName,
			VideoCodec: vStream.CodecName,
			AudioCodec: aStream.CodecName,
		},
	})
	return
}

func addStream(fullPath string, info scanFileInfo) (err error) {
	relPath := strings.TrimPrefix(fullPath, mediaDir)
	slog.Debug("> STREAM", "path", relPath)

	var f *os.File
	if f, err = os.Open(path.Join(fullPath, "stream.json")); err != nil {
		return
	}
	var meta *StreamMeta
	jsonParser := json.NewDecoder(f)
	if err = jsonParser.Decode(&meta); err != nil {
		return
	}
	if len(meta.Command) == 0 {
		err = fmt.Errorf("no command found in '%s'", fullPath)
		return
	}

	var objectID string
	var parentID string

	if objectID, parentID, err = generateNewObjectId(relPath); err != nil {
		return
	}

	err = insert("OBJECTS", map[string]any{
		"OBJECT_ID": objectID,
		"PARENT_ID": parentID,
		"PATH":      relPath,
		"TYPE":      TypeStream,
		"TIMESTAMP": info.tm.Unix(),
		"SIZE":      info.size,
		"META":      meta,
	})
	return
}
