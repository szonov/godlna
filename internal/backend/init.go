package backend

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/szonov/godlna/internal/fs_utils"
	"path"
	"path/filepath"
)

func Init(media, cache string) (err error) {
	if MediaDir, err = filepath.Abs(media); err != nil {
		return
	}
	if CacheDir, err = filepath.Abs(cache); err != nil {
		return
	}
	if err = fs_utils.EnsureDirectoryExists(MediaDir); err != nil {
		return
	}
	if err = fs_utils.EnsureDirectoryExists(CacheDir); err != nil {
		return
	}
	if DB, err = sql.Open("sqlite3", path.Join(CacheDir, "db.sqlite")); err != nil {
		return
	}
	err = createSchema()
	return
}
