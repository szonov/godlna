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
	if _, err = DB.Exec(`UPDATE OBJECTS SET TO_DELETE = 1 WHERE OBJECT_ID <> '0'`); err != nil {
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
func (s *Scanner) readDir(dir string) error {
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
			slog.Debug("skip entry", "path", fullPath, "err", err.Error())
			continue
		}

		// folder
		if entry.IsDir() {
			if err = s.checkFolder(fullPath, info); err != nil {
				slog.Info("directory add problem", "path", fullPath, "err", err.Error())
			} else if err = s.readDir(fullPath); err != nil {
				slog.Info("directory read problem", "path", fullPath, "err", err.Error())
			}
			continue
		}

		// video
		if isVideoFile(entry.Name()) {
			if err = s.checkVideo(fullPath, info); err != nil {
				slog.Info("video file add problem", "path", fullPath, "err", err.Error())
			}
		}
	}
	return nil
}

func (s *Scanner) checkFolder(fullPath string, info fs.FileInfo) (err error) {
	var relPath string
	if relPath, err = relativePath(fullPath); err != nil {
		return
	}
	slog.Debug("FOLDER", "path", relPath)

	var res sql.Result
	var affectedCount int64
	query := `UPDATE OBJECTS SET TO_DELETE = 0 WHERE PATH = ? AND TYPE = ?`
	if res, err = DB.Exec(query, relPath, Folder); err != nil {
		return
	}
	if affectedCount, err = res.RowsAffected(); err != nil || affectedCount > 0 {
		// one from: [record exists, and we removed mark TO_DELETE to this record] or [error in operation]
		return
	}

	var parentID string
	if parentID, err = findParentId(relPath); err != nil {
		return
	}

	_, err = sq.Insert("OBJECTS").SetMap(map[string]any{
		"OBJECT_ID": getNewObjectId(parentID),
		"PARENT_ID": parentID,
		"TYPE":      Folder,
		"TITLE":     info.Name(),
		"PATH":      relPath,
		"UPDATE_ID": s.newUpdateID,
		"TO_DELETE": 0,
	}).RunWith(DB).Exec()

	err = s.incrementChildrenCount(err, parentID)
	return
}

func (s *Scanner) checkVideo(fullPath string, info fs.FileInfo) (err error) {
	var relPath string
	if relPath, err = relativePath(fullPath); err != nil {
		return
	}
	slog.Debug("> VIDEO", "path", relPath)

	var res sql.Result
	var affectedCount int64
	query := `UPDATE OBJECTS SET TO_DELETE = 0 WHERE PATH = ? AND TYPE = ?`
	if res, err = DB.Exec(query, relPath, Video); err != nil {
		return err
	}
	if affectedCount, err = res.RowsAffected(); err != nil || affectedCount > 0 {
		// one from: [record exists, and we removed mark TO_DELETE to this record] or [error in operation]
		return
	}

	var parentID string
	if parentID, err = findParentId(relPath); err != nil {
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

	var b []byte
	if b, err = json.MarshalIndent(ffdata, "", "  "); err != nil {
		err = fmt.Errorf("JSON Marshal '%s' : %w", fullPath, err)
		return
	}
	slog.Debug("FFPROBE", "json", "\n"+string(b))

	size, _ := strconv.ParseUint(ffdata.Format.Size, 10, 64)
	sampleRate, _ := strconv.ParseUint(aStream.SampleRate, 10, 64)
	var bitrate uint64
	if bitrate, err = strconv.ParseUint(ffdata.Format.BitRate, 10, 64); err == nil {
		if bitrate > 8 {
			bitrate = bitrate / 8
		}
	}

	_, err = sq.Insert("OBJECTS").SetMap(map[string]any{
		"OBJECT_ID":    getNewObjectId(parentID),
		"PARENT_ID":    parentID,
		"TYPE":         Video,
		"TITLE":        NameWithoutExt(info.Name()),
		"PATH":         relPath,
		"TIMESTAMP":    info.ModTime().Unix(),
		"UPDATE_ID":    s.newUpdateID,
		"DURATION_SEC": ffdata.Format.DurationSeconds,
		"SIZE":         size,
		"RESOLUTION":   fmt.Sprintf("%dx%d", vStream.Width, vStream.Height),
		"CHANNELS":     aStream.Channels,
		"SAMPLE_RATE":  sampleRate,
		"BITRATE":      bitrate,
		"MIME":         detectMime(path.Ext(relPath), vStream, aStream),
		"TO_DELETE":    0,
	}).RunWith(DB).Exec()

	err = s.incrementChildrenCount(err, parentID)
	return
}

func (s *Scanner) incrementChildrenCount(err error, objectID string) error {
	if err != nil {
		return err
	}
	query := `UPDATE OBJECTS SET SIZE = SIZE + 1, UPDATE_ID = ?  where OBJECT_ID = ? AND TYPE = ?`
	_, err = DB.Exec(query, s.newUpdateID, objectID, Folder)
	return err
}

func GetCurrentUpdateID() uint64 {
	var updateId uint64
	err := DB.QueryRow(`SELECT VALUE FROM SETTINGS WHERE KEY = 'UPDATE_ID'`).Scan(&updateId)
	if err != nil {
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

func findParentId(relPath string) (parentID string, err error) {
	folder := filepath.Dir(relPath)
	if folder == "/" {
		parentID = "0"
		return
	}
	query := `SELECT OBJECT_ID FROM OBJECTS WHERE PATH = ? AND TYPE = ?`
	err = DB.QueryRow(query, folder, Folder).Scan(&parentID)
	return
}

func relativePath(fullPath string) (string, error) {
	if strings.HasPrefix(fullPath, MediaDir) {
		return strings.TrimPrefix(fullPath, MediaDir), nil
	}
	return "", fmt.Errorf("%s is not in MediaDir folder", fullPath)
}

func detectMime(ext string, vStream *ffprobe.Stream, aStream *ffprobe.Stream) string {
	if strings.Contains(vStream.CodecName, "matroska") {
		return "video/x-matroska"
	}
	switch ext {
	case ".avi":
		return "video/avi"
	default:
		return "video/x-msvideo"
	}
}
