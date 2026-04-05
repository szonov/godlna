package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type BookmarkInfo struct {
	Bookmark sql.NullInt64
}

func (bmi *BookmarkInfo) readCacheFile(file string) error {
	body, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// no file - it is OK.
			bmi.Bookmark = sql.NullInt64{}
			return nil
		}
		// other error should be reported
		return fmt.Errorf("failed to read: %w", err)
	}

	value := strings.TrimSpace(string(body))
	if value == "" {
		bmi.Bookmark = sql.NullInt64{}
		return nil
	}

	s, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parseInt: %w", err)
	}
	if s < 0 {
		return fmt.Errorf("wrong bookmark: is negative")
	}

	bmi.Bookmark = sql.NullInt64{Int64: s, Valid: true}
	return nil
}

func (bmi *BookmarkInfo) writeCacheFile(file string) error {
	// valid, not empty value - should write file
	if bmi != nil && bmi.Bookmark.Valid {

		// create directory for file
		if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create dir: %w", err)
		}

		body := []byte(strconv.FormatInt(bmi.Bookmark.Int64, 10))

		// write file
		if err := os.WriteFile(file, body, 0666); err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}
		return nil
	}

	// invalid value, should delete file if exists
	_ = os.Remove(file)
	if _, err := os.Stat(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func bookmarkInfoCacheFile(videoFile string) string {
	return filepath.Dir(videoFile) + "/@eaDir/" + filepath.Base(videoFile) + "/GODLNA_BOOKMARK"
}

func GetBookmarkInfo(videoFile string) (*BookmarkInfo, error) {
	cacheFile := bookmarkInfoCacheFile(videoFile)

	bmi := new(BookmarkInfo)
	err := bmi.readCacheFile(cacheFile)

	return bmi, err
}

func SetBookmarkInfo(videoFile string, bmi *BookmarkInfo) error {
	cacheFile := bookmarkInfoCacheFile(videoFile)

	return bmi.writeCacheFile(cacheFile)
}
