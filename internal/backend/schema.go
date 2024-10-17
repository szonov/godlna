package backend

import (
	"database/sql"
	"errors"
)

func createSchema() (err error) {

	err = execQuery(err, `CREATE TABLE IF NOT EXISTS OBJECTS (
			ID 				INTEGER PRIMARY KEY AUTOINCREMENT,

			-- common properties
			OBJECT_ID 	TEXT UNIQUE NOT NULL,
			PARENT_ID 	TEXT NOT NULL,
			TYPE 		INTEGER NOT NULL, -- 1: folder, 2: video item
			PATH 		TEXT NOT NULL,
			SIZE 		INTEGER NOT NULL default 0, -- children count for container, file size for video item

			-- video item property
			TIMESTAMP 		INTEGER,
			RESOLUTION 		TEXT,
			CHANNELS   		INTEGER,
			SAMPLE_RATE		INTEGER,
			BITRATE         INTEGER,
			BOOKMARK 		INTEGER,
			DURATION 		INTEGER,
			FORMAT			TEXT,
			VIDEO_CODEC		TEXT,
			AUDIO_CODEC		TEXT,

			-- system use properties
			TO_DELETE 	INTEGER DEFAULT 0,
			LEVEL 		INTEGER NOT NULL
		)`)

	var r_ string
	err = DB.QueryRow("SELECT OBJECT_ID FROM OBJECTS WHERE OBJECT_ID = '0'").Scan(&r_)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		err = insertObject(map[string]any{
			"OBJECT_ID": "0",
			"PARENT_ID": "-1",
			"PATH":      "/",
			"TYPE":      Folder,
		})
	}
	return
}
