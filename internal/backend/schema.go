package backend

import (
	"time"
)

const (
	MusicID = "1"
	VideoID = "2"
	ImageID = "3"

	ClassFolder = "container.storageFolder"
	ClassVideo  = "item.videoItem"
)

func createSchema() (err error) {

	version := 0

	err = execQuery(err, `CREATE TABLE IF NOT EXISTS SETTINGS (KEY TEXT UNIQUE NOT NULL, VALUE TEXT)`)

	if err == nil {
		_ = DB.QueryRow(`SELECT VALUE FROM SETTINGS WHERE KEY = 'VERSION'`).Scan(&version)
	}

	if version == 0 {

		// create objects table
		err = execQuery(err, `CREATE TABLE IF NOT EXISTS OBJECTS (
			ID INTEGER PRIMARY KEY AUTOINCREMENT,
			OBJECT_ID TEXT UNIQUE NOT NULL,
			PARENT_ID TEXT NOT NULL,
			CLASS TEXT NOT NULL,
			TITLE TEXT COLLATE NOCASE,
			TIMESTAMP INTEGER,
			CHILDREN_COUNT INTEGER DEFAULT 0,
			PATH TEXT DEFAULT NULL,
			META_DATA TEXT,
			UPDATE_ID INTEGER,
			BOOKMARK INTEGER default 0,
			TO_DELETE INTEGER DEFAULT 0
		)`)

		query := `INSERT INTO OBJECTS 
    			  (TITLE, OBJECT_ID, PARENT_ID, CLASS, UPDATE_ID, CHILDREN_COUNT, TIMESTAMP)
				  VALUES (?, ?, ?, ?, ?, ?, ?)`
		tms := time.Now().Unix()

		err = execQuery(err, query, "root", "0", "-1", ClassFolder, "1", "3", tms)
		err = execQuery(err, query, "Music", MusicID, "0", ClassFolder, "1", "0", tms)
		err = execQuery(err, query, "Video", VideoID, "0", ClassFolder, "1", "0", tms)
		err = execQuery(err, query, "Images", ImageID, "0", ClassFolder, "1", "0", tms)

		err = execQuery(err, `INSERT INTO SETTINGS (KEY, VALUE) VALUES ('UPDATE_ID', '10')`)
		err = execQuery(err, `INSERT INTO SETTINGS (KEY, VALUE) VALUES ('VERSION', '1')`)
	}
	return
}

func execQuery(err error, query string, args ...interface{}) error {
	if err != nil {
		return err
	}
	_, err = DB.Exec(query, args...)
	return err
}
