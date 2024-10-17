package backend

func createSchema() (err error) {

	err = execQuery(err, `CREATE TABLE IF NOT EXISTS SETTINGS (KEY TEXT UNIQUE NOT NULL, VALUE TEXT)`)
	err = execQuery(err, `CREATE TABLE IF NOT EXISTS OBJECTS (
			ID 				INTEGER PRIMARY KEY AUTOINCREMENT,

			-- common properties
			OBJECT_ID 	TEXT UNIQUE NOT NULL,
			PARENT_ID 	TEXT NOT NULL,
			TYPE 		INTEGER NOT NULL, -- 1: folder, 2: video item
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
			DURATION 		INTEGER,
			FORMAT			TEXT,
			VIDEO_CODEC		TEXT,
			AUDIO_CODEC		TEXT,

			-- system use properties
			TO_DELETE 	INTEGER DEFAULT 0
		)`)

	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM OBJECTS").Scan(&count)

	if err == nil && count == 0 {
		query := `INSERT INTO OBJECTS (OBJECT_ID, PARENT_ID, PATH,  TYPE, UPDATE_ID) VALUES (?, ?, ?, ?, ?)`
		err = execQuery(err, query, "0", "-1", "/", Folder, "120")
		err = execQuery(err, `INSERT INTO SETTINGS (KEY, VALUE) VALUES ('UPDATE_ID', '120')`)
	}

	return
}
