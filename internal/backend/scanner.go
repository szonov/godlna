package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/szonov/godlna/internal/fs_utils"

	sq "github.com/Masterminds/squirrel"
	"gopkg.in/vansante/go-ffprobe.v2"
)

type (
	Scanner struct {
		lastUpdateID UpdateIdNumber
		newUpdateID  UpdateIdNumber
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
	s.lastUpdateID = GetSystemUpdateId()
	s.newUpdateID = s.lastUpdateID + 1
	return
}

func (s *Scanner) afterScan() (err error) {
	for {
		if !s.clearOneDeletedFolder() {
			break
		}
	}

	slog.Debug("After scan media dir", "UPDATE_ID", s.newUpdateID)

	if newUpdateId := getMaxUpdateID(); newUpdateId > s.lastUpdateID {
		setUpdateID(newUpdateId)
		s.lastUpdateID = newUpdateId
		s.newUpdateID = newUpdateId
	} else {
		s.newUpdateID = s.lastUpdateID
	}

	// todo: delete marked TO_DELETE

	return
}

// clearOneDeletedFolder and returns bool:
// - true - folder deleted, required to run again this method to delete next folder
// - false - error happens or no more folders to delete, mean stop folder deletion
func (s *Scanner) clearOneDeletedFolder() bool {
	var objectID string
	var query string
	var err error

	// select first folder
	query = `SELECT OBJECT_ID FROM OBJECTS WHERE TO_DELETE = 1 AND TYPE = ? ORDER BY ID LIMIT 1`
	if err = DB.QueryRow(query, Folder).Scan(&objectID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("select folder to_delete", "err", err.Error())
		}
		return false
	}

	// delete folder and its children from database
	query = `DELETE FROM OBJECTS WHERE OBJECT_ID = ? OR OBJECT_ID like ?`
	if _, err = DB.Exec(query, objectID, objectID+"$%"); err != nil {
		slog.Error("remove folder to_delete", "err", err.Error(), "objectID", objectID)
		return false
	}
	// remove thumbnails cache of object and children
	removeThumbnails(objectID)

	// increment parent's updateID
	touchParent(objectID, s.newUpdateID)

	return true
}
func (s *Scanner) clearAllDeletedFiles() {
	var query string
	var err error
	var rows *sql.Rows

	query = `SELECT OBJECT_ID, PARENT_ID FROM OBJECTS WHERE TO_DELETE = 1 AND TYPE = ?`
	rows, err = DB.Query(query, Video)
	if err != nil {
		slog.Error("clear video to_delete", "err", err.Error())
		return
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			slog.Error("clear video to_delete :: rows.Close", "err", err.Error())
		}
	}(rows)

	for rows.Next() {
		var (
			objectID string
			parentID string
		)

		if err = rows.Scan(&objectID, &parentID); err != nil {
			slog.Error("clear video to_delete :: rows.Scan", "err", err.Error())
			return
		}
	}
	//if !rows.NextResultSet() {
	//	log.Fatalf("expected more result sets: %v", rows.Err())
	//}
	//var roleMap = map[int64]string{1: "user", 2: "admin", 3: "gopher"}
	//for rows.Next() {
	//	var (
	//		id int64 role int64
	//	)
	//	if err := rows.Scan(&id, &role); err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Printf("id %d has role %s\n", id, roleMap[role])
	//}
	//if err := rows.Err(); err != nil {
	//	log.Fatal(err)
	//}
	//.Scan(&objectID)
	//err != nil{
	//	if !errors.Is(err, sql.ErrNoRows){
	//	slog.Error("select folder to_delete", "err", err.Error())
	//}
	//	return false
	//}
}

func removeObjectWithChildren(objectID string) (err error) {
	query := `DELETE FROM OBJECTS WHERE OBJECT_ID = ? OR OBJECT_ID like ?`
	if _, err = DB.Exec(query, objectID, objectID+"$%"); err != nil {
		slog.Error("removeObjectWithChildren", "err", err.Error(), "objectID", objectID)
	}
	removeThumbnails(objectID)
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
			if err = s.checkFolder(fullPath); err != nil {
				slog.Info("directory add problem", "path", fullPath, "err", err.Error())
			} else if err = s.readDir(fullPath); err != nil {
				slog.Info("directory read problem", "path", fullPath, "err", err.Error())
			}
			continue
		}

		// video
		if fs_utils.IsVideoFile(entry.Name()) {
			if err = s.checkVideo(fullPath, info); err != nil {
				slog.Info("video file add problem", "path", fullPath, "err", err.Error())
			}
		}
	}
	return nil
}

func (s *Scanner) checkFolder(fullPath string) (err error) {
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
		"PATH":      relPath,
		"UPDATE_ID": s.newUpdateID,
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

	size, _ := strconv.ParseUint(ffdata.Format.Size, 10, 64)
	sampleRate, _ := strconv.ParseUint(aStream.SampleRate, 10, 64)
	var bitrate uint64
	if bitrate, err = strconv.ParseUint(ffdata.Format.BitRate, 10, 64); err == nil {
		if bitrate > 8 {
			bitrate = bitrate / 8
		}
	}

	_, err = sq.Insert("OBJECTS").SetMap(map[string]any{
		"OBJECT_ID":   getNewObjectId(parentID),
		"PARENT_ID":   parentID,
		"TYPE":        Video,
		"PATH":        relPath,
		"TIMESTAMP":   info.ModTime().Unix(),
		"UPDATE_ID":   s.newUpdateID,
		"DURATION":    uint64(ffdata.Format.DurationSeconds),
		"SIZE":        size,
		"RESOLUTION":  fmt.Sprintf("%dx%d", vStream.Width, vStream.Height),
		"CHANNELS":    aStream.Channels,
		"SAMPLE_RATE": sampleRate,
		"BITRATE":     bitrate,
		"FORMAT":      ffdata.Format.FormatName,
		"VIDEO_CODEC": vStream.CodecName,
		"AUDIO_CODEC": aStream.CodecName,
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

func GetSystemUpdateId() UpdateIdNumber {
	var updateId UpdateIdNumber
	err := DB.QueryRow(`SELECT VALUE FROM SETTINGS WHERE KEY = 'UPDATE_ID'`).Scan(&updateId)
	if err != nil {
		slog.Error("select UPDATE_ID", "err", err.Error())
		return 1
	}
	return updateId
}

func getMaxUpdateID() UpdateIdNumber {
	var updateID UpdateIdNumber
	if err := DB.QueryRow(`SELECT MAX(UPDATE_ID) FROM OBJECTS`).Scan(&updateID); err != nil {
		slog.Error("select MAX(UPDATE_ID)", "err", err.Error())
		return 1
	}
	return updateID
}

func setUpdateID(updateID UpdateIdNumber) {
	_, err := DB.Exec(`UPDATE SETTINGS SET VALUE = ? WHERE KEY = 'UPDATE_ID'`, updateID)
	if err != nil {
		slog.Error("update settings", "err", err.Error())
	}
}

func getNewObjectId(parentID string) string {
	var seq int64
	if err := DB.QueryRow(`SELECT seq FROM SQLITE_SEQUENCE WHERE name = 'OBJECTS'`).Scan(&seq); err != nil {
		slog.Error("select seq", "err", err.Error())
	}
	seq += 1

	return parentID + "$" + strconv.FormatInt(seq, 10)
}

func findParentId(relPath string) (parentID string, err error) {
	query := `SELECT OBJECT_ID FROM OBJECTS WHERE PATH = ? AND TYPE = ?`
	err = DB.QueryRow(query, filepath.Dir(relPath), Folder).Scan(&parentID)
	return
}

func GetParentID(objectID string) string {
	i := strings.LastIndex(objectID, "$")
	if i > -1 {
		return objectID[:i]
	}
	if objectID == "0" {
		return "-1"
	}
	return "0"
}

func relativePath(fullPath string) (string, error) {
	if strings.HasPrefix(fullPath, MediaDir) {
		return strings.TrimPrefix(fullPath, MediaDir), nil
	}
	return "", fmt.Errorf("%s is not in MediaDir folder", fullPath)
}
