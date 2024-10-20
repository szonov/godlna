package scanner

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/szonov/godlna/internal/ffmpeg"
	"github.com/szonov/godlna/internal/fs_utils"
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

type Scanner struct {
	root           string
	cacheLifeTime  time.Duration
	guard          *Guard
	db             *sql.DB
	OnObjectDelete func(objectID string)
}

type cachedObject struct {
	oid   string
	class string
	tm    *types.NullableNumber
	size  *types.NullableNumber
}

func NewScanner(root string, cacheLifeTime time.Duration, db *sql.DB) *Scanner {
	return &Scanner{
		root:          root,
		guard:         NewGuard(),
		cacheLifeTime: cacheLifeTime,
		db:            db,
	}
}

func (ds *Scanner) EnsureObjectIsUpToDate(objectID string) {
	if ds.guard.TryLock(objectID) {
		if outdated, dir := ds.outdatedInfo(objectID); outdated {
			ds.rescanDir(path.Join(ds.root, dir), objectID)
		}
		ds.guard.Unlock(objectID)
	} else {
		slog.Warn("CAN NOT GET LOCK", "OBJECT_ID", objectID)
	}
}

func (ds *Scanner) outdatedInfo(objectID string) (outdated bool, dir string) {
	var class string
	var tm *types.NullableNumber
	outdated = false

	err := ds.db.QueryRow("SELECT PATH, CLASS, TIMESTAMP FROM OBJECTS WHERE OBJECT_ID = ?", objectID).
		Scan(&dir, &class, &tm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && objectID == "0" {
			outdated = true
			dir = "/"
			ds.createRootObject()
		} else {
			slog.Error("needRescan", "OBJECT_ID", objectID, "err", err.Error())
		}
		return
	}
	if class == "container.storageFolder" {
		if tm == nil || tm.Time().Add(ds.cacheLifeTime).Before(time.Now()) {
			outdated = true
			return
		}
	}
	return
}

func (ds *Scanner) createRootObject() {
	_ = ds.insertObject(map[string]any{
		"OBJECT_ID": "0",
		"PARENT_ID": "-1",
		"PATH":      "/",
		"CLASS":     "container.storageFolder",
	})
}

func (ds *Scanner) rescanDir(dir, objectID string) {
	slog.Debug("rescanDir", "OBJECT_ID", objectID, "dir", dir)
	oldChildren := ds.cachedChildren(objectID)

	var entries []fs.DirEntry
	var info fs.FileInfo
	var err error

	if entries, err = os.ReadDir(dir); err != nil {
		ds.deleteObject(objectID)
		ds.deleteOldChildren(oldChildren)
		return
	}

	for _, entry := range entries {
		fullPath := path.Join(dir, entry.Name())
		if entry.IsDir() && entry.Name() == "@eaDir" {
			// synology special folders
			continue
		}

		//can not get info about file/directory - just skip it
		if info, err = entry.Info(); err != nil {
			slog.Error("entry.Info", "path", fullPath, "err", err.Error())
			continue
		}

		if entry.IsDir() {
			if oldObject, exists := oldChildren[fullPath]; exists {
				// same folder?
				if oldObject.class == "container.storageFolder" {
					delete(oldChildren, fullPath)
					continue
				}
			}
			if err = ds.addFolder(fullPath); err != nil {
				slog.Error("addFolder", "path", fullPath, "err", err.Error())
				continue
			}
		} else if fs_utils.IsVideoFile(entry.Name()) {
			if oldObject, exists := oldChildren[fullPath]; exists {
				// same video file? (size and timestamp the same)
				if oldObject.class == "item.videoItem" && oldObject.tm.Int64() == info.ModTime().Unix() && info.Size() == oldObject.size.Int64() {
					delete(oldChildren, fullPath)
					continue
				}
			}
			if err = ds.addVideo(fullPath, info.ModTime()); err != nil {
				slog.Error("addVideo", "path", fullPath, "err", err.Error())
				continue
			}
		}
	}

	ds.deleteOldChildren(oldChildren)
	ds.setJustUpdated(objectID)
}

func (ds *Scanner) setJustUpdated(objectID string) {
	_, err := ds.db.Exec(`UPDATE OBJECTS SET TIMESTAMP = ? WHERE OBJECT_ID =?`, time.Now().Unix(), objectID)
	if err != nil {
		slog.Error("setJustUpdated", "OBJECT_ID", objectID, "err", err.Error())
	}
}

func (ds *Scanner) cachedChildren(objectID string) map[string]cachedObject {
	m := map[string]cachedObject{}
	q := "SELECT OBJECT_ID, PATH, CLASS, TIMESTAMP, SIZE FROM OBJECTS WHERE PARENT_ID = ?"
	rows, err := ds.db.Query(q, objectID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("cachedObjects", "OBJECT_ID", objectID, "error", err.Error())
		}
		return m
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			slog.Error("cachedObjects.close", "OBJECT_ID", objectID, "err", err.Error())
		}
	}(rows)

	for rows.Next() {
		var p, oid, class string
		var tm, size *types.NullableNumber
		if err = rows.Scan(&oid, &p, &class, &tm, &size); err != nil {
			slog.Error("cachedObjects.scan", "OBJECT_ID", objectID, "err", err.Error())
			return m
		}
		m[path.Join(ds.root, p)] = cachedObject{oid: oid, class: class, tm: tm, size: size}
	}
	return m
}

func (ds *Scanner) deleteOldChildren(objects map[string]cachedObject) {
	for _, o := range objects {
		ds.deleteObject(o.oid)
	}
}

func (ds *Scanner) deleteObject(objectID string) {
	if objectID != "0" {
		_, err := ds.db.Exec("DELETE FROM OBJECTS WHERE OBJECT_ID = ? OR OBJECT_ID LIKE ?", objectID, objectID+"$%")
		if err != nil {
			slog.Error("deleteObject", "OBJECT_ID", objectID, "error", err.Error())
		} else if ds.OnObjectDelete != nil {
			ds.OnObjectDelete(objectID)
		}
	}
}

func (ds *Scanner) addFolder(fullPath string) (err error) {
	relPath := strings.TrimPrefix(fullPath, ds.root)

	slog.Debug("FOLDER", "path", relPath)

	var objectID string
	var parentID string

	if objectID, parentID, err = ds.generateNewObjectId(relPath); err != nil {
		return
	}

	err = ds.insertObject(map[string]any{
		"OBJECT_ID": objectID,
		"PARENT_ID": parentID,
		"CLASS":     "container.storageFolder",
		"PATH":      relPath,
	})

	return
}

func (ds *Scanner) addVideo(fullPath string, modTime time.Time) (err error) {
	relPath := strings.TrimPrefix(fullPath, ds.root)

	slog.Debug("> VIDEO", "path", relPath)

	var objectID string
	var parentID string

	if objectID, parentID, err = ds.generateNewObjectId(relPath); err != nil {
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

	err = ds.insertObject(map[string]any{
		"OBJECT_ID":   objectID,
		"PARENT_ID":   parentID,
		"CLASS":       "item.videoItem",
		"PATH":        relPath,
		"TIMESTAMP":   modTime.Unix(),
		"DURATION":    ffData.Format.Duration().Milliseconds(),
		"SIZE":        ffData.Format.Size,
		"RESOLUTION":  vStream.Resolution(),
		"CHANNELS":    aStream.Channels,
		"SAMPLE_RATE": aStream.SampleRate,
		"BITRATE":     ffData.Format.BitRate(),
		"FORMAT":      ffData.Format.FormatName,
		"VIDEO_CODEC": vStream.CodecName,
		"AUDIO_CODEC": aStream.CodecName,
	})
	return
}

func (ds *Scanner) generateNewObjectId(relPath string) (objectID, parentID string, err error) {
	query := `SELECT OBJECT_ID FROM OBJECTS WHERE PATH = ? AND CLASS = 'container.storageFolder'`
	err = ds.db.QueryRow(query, filepath.Dir(relPath)).Scan(&parentID)
	if err == nil {
		var seq int64
		err = ds.db.QueryRow(`SELECT seq FROM SQLITE_SEQUENCE WHERE name = 'OBJECTS'`).Scan(&seq)
		if err == nil {
			return parentID + "$" + strconv.FormatInt(seq+1, 10), parentID, nil
		}
	}
	return "", "", err
}

// insertObject adds new object to database
func (ds *Scanner) insertObject(data map[string]any) error {
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
	q := "INSERT INTO" + " OBJECTS (" + strings.Join(c, ",") + ") VALUES (" + strings.Join(m, ",") + ")"
	if _, err := ds.db.Exec(q, args...); err != nil {
		return err
	}
	return nil
}
