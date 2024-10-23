package db

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/szonov/godlna/internal/fs_utils"
	"os"
	"path"
	"path/filepath"
	"time"
)

// Init initialize package for following usage
// mediaDirectory - is the full path to media directory
// cacheDirectory - is the full path to cache directory (when empty $HOME/.local/godlna is used)
// objectCacheLifeTime - during this time, the directory will not be rescanned.
func Init(mediaDirectory, cacheDirectory string, objectCacheLifeTime time.Duration) (err error) {
	// cache directory
	if cacheDirectory == "" {
		cacheDirectory, err = os.UserHomeDir()
		if err != nil {
			return
		}
		cacheDirectory = path.Join(cacheDirectory, ".local", "godlna")
	}
	if cacheDir, err = filepath.Abs(cacheDirectory); err != nil {
		return
	}
	if err = fs_utils.EnsureDirectoryExists(cacheDir); err != nil {
		return
	}
	// media directory
	if mediaDir, err = filepath.Abs(mediaDirectory); err != nil {
		return
	}
	if !fs_utils.FileExists(mediaDir) {
		err = fmt.Errorf("media directory '%s' does not exist", mediaDir)
		return
	}
	// database
	if db, err = sql.Open("sqlite3", path.Join(cacheDir, "db.sqlite")); err != nil {
		return
	}
	// scan parameters
	cacheLifeTime = objectCacheLifeTime
	scanGuard = NewGuard()
	// database schema
	err = createSchema()
	EnsureObjectIsUpToDate("0")
	return
}

func createSchema() (err error) {
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS OBJECTS (
			ID 			INTEGER PRIMARY KEY AUTOINCREMENT,
			OBJECT_ID 	TEXT UNIQUE NOT NULL,
			PARENT_ID 	TEXT NOT NULL,
			PATH 		TEXT NOT NULL,
			TYPE 		INTEGER NOT NULL,
			TIMESTAMP 	INTEGER, -- file modification time for files, last scan time for dirs
			SIZE	 	INTEGER NOT NULL DEFAULT 0, -- filesize for files, not used count for dirs
			BOOKMARK 	INTEGER,
			META 		TEXT
		)`)
	return
}
