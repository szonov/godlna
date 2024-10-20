package store

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/szonov/godlna/internal/fs_utils"
	"github.com/szonov/godlna/internal/store/scanner"
	"path"
	"path/filepath"
	"time"
)

var (
	mediaDir         string
	cacheDir         string
	db               *sql.DB
	directoryScanner *scanner.Scanner
)

func Init(media, cache string, cacheLifeTime time.Duration) (err error) {
	if mediaDir, err = filepath.Abs(media); err != nil {
		return
	}
	if cacheDir, err = filepath.Abs(cache); err != nil {
		return
	}
	if err = fs_utils.EnsureDirectoryExists(mediaDir); err != nil {
		return
	}
	if err = fs_utils.EnsureDirectoryExists(cacheDir); err != nil {
		return
	}
	if db, err = sql.Open("sqlite3", path.Join(cacheDir, "db.sqlite")); err != nil {
		return
	}
	err = createSchema()
	directoryScanner = scanner.NewScanner(mediaDir, cacheLifeTime, db)
	directoryScanner.OnObjectDelete = removeObjectCache
	return
}

func createSchema() (err error) {

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS OBJECTS (
			ID 				INTEGER PRIMARY KEY AUTOINCREMENT,

			-- common properties
			OBJECT_ID 	TEXT UNIQUE NOT NULL,
			PARENT_ID 	TEXT NOT NULL,
			CLASS 		TEXT NOT NULL,
			PATH 		TEXT NOT NULL,

			-- video item property
			TIMESTAMP 		INTEGER,
			SIZE	 		INTEGER,
			RESOLUTION 		TEXT,
			CHANNELS   		INTEGER,
			SAMPLE_RATE		INTEGER,
			BITRATE         INTEGER,
			BOOKMARK 		INTEGER,
			DURATION 		INTEGER,
			FORMAT			TEXT,
			VIDEO_CODEC		TEXT,
			AUDIO_CODEC		TEXT
		)`)
	return
}
