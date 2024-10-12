package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gopkg.in/vansante/go-ffprobe.v2"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strconv"
	"time"
)

type (
	Scanner struct {
	}

	scanObject struct {
		path     string
		info     fs.FileInfo
		objectId string
		parentID string
		title    string
	}
)

func (so *scanObject) exists() bool {
	return so.objectId != ""
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) newScanObject(path string, info fs.FileInfo) (*scanObject, error) {
	title := info.Name()
	if !info.IsDir() {
		title = nameWithoutExt(title)
	}

	obj := &scanObject{
		path:  path,
		info:  info,
		title: title,
	}
	var err error
	var query string
	var row *sql.Row
	query = `SELECT o.OBJECT_ID, o.PARENT_ID from OBJECTS o, DETAILS d WHERE o.DETAIL_ID = d.ID AND d.PATH = ?`
	row = DB.QueryRow(query, path)
	if err = row.Scan(&obj.objectId, &obj.parentID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// no record - search parent in db by directory name
	if obj.parentID == "" {
		folder := filepath.Dir(path)
		if folder == MediaDir {
			obj.parentID = VideoID
		} else {
			query = `SELECT o.OBJECT_ID from OBJECTS o, DETAILS d WHERE o.DETAIL_ID = d.ID AND o.CLASS = ? AND d.PATH = ?`
			row = DB.QueryRow(query, ClassFolder, folder)
			if err = row.Scan(&obj.parentID); err != nil {
				return nil, err
			}
		}
	}

	return obj, nil
}

func (s *Scanner) newFolder(so *scanObject) (err error) {
	obj := &Object{
		Class:    ClassFolder,
		ParentID: so.parentID,
		Name:     so.title,
		Details: &Details{
			Path: so.path,
		},
	}
	if err = obj.Save(); err != nil {
		err = fmt.Errorf("folder '%s' : %w", so.path, err)
	}
	return
}

func (s *Scanner) newVideo(so *scanObject) (err error) {
	obj := &Object{
		Class:    ClassItemVideo,
		ParentID: so.parentID,
		Name:     so.title,
		Details: &Details{
			Path: so.path,
		},
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	var ffdata *ffprobe.ProbeData
	ffdata, err = ffprobe.ProbeURL(ctx, so.path)
	if err != nil {
		err = fmt.Errorf("ffprobe '%s' : %w", so.path, err)
		return
	}

	vstream := ffdata.FirstVideoStream()
	astream := ffdata.FirstAudioStream()

	if vstream == nil {
		err = fmt.Errorf("no video stream '%s' (%s) : %w", so.path, ffdata.Format.Size, err)
		return
	}
	if astream == nil {
		err = fmt.Errorf("no audio stream '%s' (%s) : %w", so.path, ffdata.Format.Size, err)
		return
	}

	var t64 int64

	// size:
	if t64, err = strconv.ParseInt(ffdata.Format.Size, 10, 64); err != nil {
		err = fmt.Errorf("size convert '%s' (%s) : %w", so.path, ffdata.Format.Size, err)
		return
	}

	obj.Details.Size = t64
	obj.Details.Timestamp = so.info.ModTime().Unix()
	obj.Details.Duration = fmtDuration(ffdata.Format.Duration())
	obj.Details.Bitrate = fmtBitrate(ffdata.Format.BitRate)
	obj.Details.SampleRate = fmtSampleRate(astream.SampleRate)

	// ffprobe -v quiet -print_format json -show_format -show_streams '/Users/zonov/tmp/dms/video-bak/Некрасивая подружка/НП.S19.E02.2023.Головоломка.mkv'

	if err = obj.Save(); err != nil {
		err = fmt.Errorf("video '%s' : %w", so.path, err)
	}
	return
}

func (s *Scanner) fileProcessor(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if path == MediaDir {
		// video
		return nil
	}

	var so *scanObject
	so, err = s.newScanObject(path, info)
	if err != nil {
		return fmt.Errorf("path '%s' : %w", path, err)
	}

	// new folder
	if info.IsDir() {
		if !so.exists() {
			return s.newFolder(so)
		}
	} else if isVideo(path) {
		if !so.exists() {
			return s.newVideo(so)
		}
	}
	return nil
}

func (s *Scanner) Scan() {
	err := filepath.Walk(MediaDir, s.fileProcessor)
	if err != nil {
		slog.Error("scan problem", "err", err.Error())
	}
}
