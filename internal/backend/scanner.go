package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"gopkg.in/vansante/go-ffprobe.v2"
)

type (
	Scanner struct {
		lastUpdateID uint64
		newUpdateID  uint64
	}
)

func NewScanner() *Scanner {
	return new(Scanner)
}

func (s *Scanner) Scan() {
	var err error
	if err = s.beforeScan(); err != nil {
		slog.Error("prepare scan problem", "err", err.Error())
		return
	}

	slog.Debug("Start scan media dir", "UPDATE_ID", s.lastUpdateID, "NEXT_UPDATE_ID", s.newUpdateID)

	if err = s.readDir(MediaDir); err != nil {
		slog.Error("scan problem", "err", err.Error())
		return
	}

	if err = s.afterScan(); err != nil {
		slog.Error("finalize scan problem", "err", err.Error())
		return
	}

	slog.Debug("Complete scan media dir", "UPDATE_ID", s.newUpdateID)
}
func (s *Scanner) beforeScan() (err error) {
	_, err = DB.Exec(`UPDATE OBJECTS SET TO_DELETE = 1 WHERE PARENT_ID <> '-1' AND PARENT_ID <> '0'`)
	if err != nil {
		return err
	}

	s.lastUpdateID = GetCurrentUpdateID()
	s.newUpdateID = s.lastUpdateID + 1

	return
}
func (s *Scanner) afterScan() (err error) {
	if newUpdateId := getMaxUpdateId(); newUpdateId > s.lastUpdateID {
		setUpdateId(newUpdateId)
		s.lastUpdateID = newUpdateId
		s.newUpdateID = newUpdateId
	} else {
		s.newUpdateID = s.lastUpdateID
	}

	// todo: delete marked TO_DELETE

	return
}
func (s *Scanner) readDir(dir string) (err error) {
	var entries []fs.DirEntry
	var info fs.FileInfo
	if entries, err = os.ReadDir(dir); err != nil {
		return
	}
	for _, entry := range entries {
		if info, err = entry.Info(); err != nil {
			return
		}
		fullPath := path.Join(dir, entry.Name())
		if entry.IsDir() {
			if err = s.checkFolder(fullPath, info); err != nil {
				return
			}
			if err = s.readDir(fullPath); err != nil {
				return
			}
		} else if isVideoFile(entry.Name()) {
			if err = s.checkVideo(fullPath, info); err != nil {
				return
			}
		}
	}
	return
}

func (s *Scanner) checkFolder(fullPath string, info fs.FileInfo) (err error) {
	slog.Debug("FOLDER", "path", fullPath)

	var res sql.Result
	var affectedCount int64
	res, err = sq.Update("OBJECTS").Set("TO_DELETE", 0).
		Where("PATH = ? AND CLASS = ?", fullPath, ClassFolder).
		RunWith(DB).Exec()

	if err != nil {
		return err
	}

	if affectedCount, err = res.RowsAffected(); err != nil || affectedCount > 0 {
		// one from: [record exists, and we removed mark TO_DELETE to this record] or [error in operation]
		return
	}

	var parentID string
	if parentID, err = findParentId(fullPath); err != nil {
		return
	}

	_, err = sq.Insert("OBJECTS").SetMap(map[string]any{
		"OBJECT_ID": getNewObjectId(parentID),
		"PARENT_ID": parentID,
		"CLASS":     ClassFolder,
		"TITLE":     info.Name(),
		"PATH":      fullPath,
		"TIMESTAMP": info.ModTime().Unix(),
		"UPDATE_ID": s.newUpdateID,
		"TO_DELETE": 0,
	}).RunWith(DB).Exec()

	if err != nil {
		query := `UPDATE OBJECTS SET CHILDREN_COUNT = CHILDREN_COUNT + 1  where OBJECT_ID = ?`
		_, err = DB.Exec(query, parentID)
	}

	return
}

func (s *Scanner) checkVideo(fullPath string, info fs.FileInfo) (err error) {
	slog.Debug("> VIDEO", "path", fullPath)

	var res sql.Result
	var affectedCount int64
	res, err = sq.Update("OBJECTS").
		SetMap(map[string]any{"TO_DELETE": 0}).
		Where(sq.Eq{"PATH": fullPath, "CLASS": ClassVideo}).
		RunWith(DB).Exec()

	if err != nil {
		return err
	}

	if affectedCount, err = res.RowsAffected(); err != nil || affectedCount > 0 {
		// one from: [record exists, and we removed mark TO_DELETE to this record] or [error in operation]
		return
	}

	var parentID string
	if parentID, err = findParentId(fullPath); err != nil {
		return
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	var ffdata *ffprobe.ProbeData
	ffdata, err = ffprobe.ProbeURL(ctx, fullPath)
	if err != nil {
		err = fmt.Errorf("ffprobe '%s' : %w", fullPath, err)
		return
	}

	var b []byte
	if b, err = json.Marshal(ffdata); err != nil {
		err = fmt.Errorf("JSON Marshal '%s' : %w", fullPath, err)
		return
	}

	_, err = sq.Insert("OBJECTS").SetMap(map[string]any{
		"OBJECT_ID":      getNewObjectId(parentID),
		"PARENT_ID":      parentID,
		"CLASS":          ClassVideo,
		"TITLE":          NameWithoutExt(info.Name()),
		"PATH":           fullPath,
		"TIMESTAMP":      info.ModTime().Unix(),
		"UPDATE_ID":      s.newUpdateID,
		"META_DATA":      string(b),
		"TO_DELETE":      0,
		"CHILDREN_COUNT": 1, // to simplify selection of object with any class which have counter > 0
	}).RunWith(DB).Exec()

	if err == nil {
		// mark container as updated and increment counter of children
		query := `UPDATE OBJECTS SET CHILDREN_COUNT = CHILDREN_COUNT + 1, UPDATE_ID = ? where OBJECT_ID = ?`
		_, err = DB.Exec(query, s.newUpdateID, parentID)
	}

	return
}

func GetCurrentUpdateID() uint64 {
	row := sq.Select("VALUE").From("SETTINGS").
		Where("KEY = ?", "UPDATE_ID").RunWith(DB).QueryRow()
	var updateId uint64
	if err := row.Scan(&updateId); err != nil {
		slog.Error("select UPDATE_ID", "err", err.Error())
		return 1
	}
	return updateId
}

func getMaxUpdateId() uint64 {
	row := sq.Select("MAX(UPDATE_ID)").From("OBJECTS").RunWith(DB).QueryRow()
	var updateId uint64
	if err := row.Scan(&updateId); err != nil {
		slog.Error("select MAX(UPDATE_ID)", "err", err.Error())
		return 1
	}
	return updateId
}

func setUpdateId(updateId uint64) {
	_, _ = sq.Update("SETTINGS").SetMap(map[string]any{"VALUE": strconv.Itoa(int(updateId))}).
		Where("KEY = ?", "UPDATE_ID").RunWith(DB).Exec()
}

func getNextAvailableId(parentID string) int64 {
	var err error
	var maxObjectID string
	query := `SELECT OBJECT_ID from OBJECTS where ID = (SELECT max(ID) from OBJECTS where PARENT_ID = ?)`
	row := DB.QueryRow(query, parentID)
	err = row.Scan(&maxObjectID)
	if err == nil {
		if p := strings.LastIndex(maxObjectID, "$"); p != -1 {
			var maxValue int64
			if maxValue, err = strconv.ParseInt(maxObjectID[p+1:], 10, 64); err == nil {
				return maxValue + 1
			}
		}
	}
	return 0
}

func getNewObjectId(parentID string) string {
	return parentID + "$" + strconv.FormatInt(getNextAvailableId(parentID), 10)
}

func findParentId(fullPath string) (parentID string, err error) {
	folder := filepath.Dir(fullPath)
	if folder == MediaDir {
		parentID = VideoID
		return
	}
	row := sq.Select("OBJECT_ID").From("OBJECTS").Where(sq.Eq{"PATH": folder, "CLASS": ClassFolder}).
		RunWith(DB).QueryRow()
	err = row.Scan(&parentID)
	return
}
