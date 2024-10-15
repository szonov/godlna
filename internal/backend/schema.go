package backend

func createSchema() (err error) {

	version := 0

	err = execQuery(err, `CREATE TABLE IF NOT EXISTS SETTINGS (KEY TEXT UNIQUE NOT NULL, VALUE TEXT)`)

	if err == nil {
		_ = DB.QueryRow(`SELECT VALUE FROM SETTINGS WHERE KEY = 'VERSION'`).Scan(&version)
	}

	if version == 0 {
		// FOLDER: objectID, parentID, Class, Title

		// create objects table
		err = execQuery(err, `CREATE TABLE IF NOT EXISTS OBJECTS (
			ID 				INTEGER PRIMARY KEY AUTOINCREMENT,

			-- common properties
			OBJECT_ID 	TEXT UNIQUE NOT NULL,
			PARENT_ID 	TEXT NOT NULL,
			TYPE 		INTEGER NOT NULL, -- 1: folder, 2: video item
			TITLE 		TEXT COLLATE NOCASE,
			PATH 		TEXT NOT NULL,
			UPDATE_ID 	INTEGER NOT NULL,
			SIZE 		INTEGER NOT NULL default 0, -- children count for container, file size for video item

			-- video item property
			TIMESTAMP 		INTEGER,
			RESOLUTION 		TEXT,
			CHANNELS   		INTEGER,
			SAMPLE_RATE		INTEGER,
			BITRATE         INTEGER,
			BOOKMARK 		INTEGER,
			DURATION_SEC 	REAL,
			MIME 			TEXT,
			META_DATA 		TEXT,

			-- system use properties
			TO_DELETE 	INTEGER DEFAULT 0
		)`)

		query := `INSERT INTO OBJECTS (TITLE, PATH, OBJECT_ID, PARENT_ID, TYPE, UPDATE_ID) VALUES (?, ?, ?, ?, ?, ?)`
		err = execQuery(err, query, "root", "/", "0", "-1", Folder, "40")

		err = execQuery(err, `INSERT INTO SETTINGS (KEY, VALUE) VALUES ('UPDATE_ID', '40')`)
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
