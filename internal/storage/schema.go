package storage

import sq "github.com/Masterminds/squirrel"

const (
	MusicID = "1"
	VideoID = "2"
	ImageID = "3"
)

func createSchema() error {

	// objects
	var query string
	var err error

	query = `
	CREATE TABLE IF NOT EXISTS OBJECTS (
	    ID INTEGER PRIMARY KEY AUTOINCREMENT, 
	    OBJECT_ID TEXT UNIQUE NOT NULL, 
	    PARENT_ID TEXT NOT NULL, 
-- 	    REF_ID TEXT DEFAULT NULL, 
	    CLASS TEXT NOT NULL, 
	    DETAIL_ID INTEGER DEFAULT NULL, 
	    NAME TEXT DEFAULT NULL
	);`

	if _, err = DB.Exec(query); err != nil {
		return err
	}

	query = `CREATE TABLE IF NOT EXISTS DETAILS (
    	ID INTEGER PRIMARY KEY AUTOINCREMENT, 
    	PATH TEXT DEFAULT NULL, 
    	SIZE INTEGER, 
    	TIMESTAMP INTEGER, 
    	TITLE TEXT COLLATE NOCASE, 
    	DURATION TEXT, 
    	BITRATE INTEGER, 
    	SAMPLERATE INTEGER, 
    	CREATOR TEXT COLLATE NOCASE, 
    	ARTIST TEXT COLLATE NOCASE, 
    	ALBUM TEXT COLLATE NOCASE, 
    	GENRE TEXT COLLATE NOCASE, 
    	COMMENT TEXT, 
    	CHANNELS INTEGER, 
    	DISC INTEGER, 
    	TRACK INTEGER, 
    	DATE DATE, 
    	RESOLUTION TEXT, 
    	THUMBNAIL BOOL DEFAULT 0, 
    	ALBUM_ART INTEGER DEFAULT 0, 
    	ROTATION INTEGER, 
    	DLNA_PN TEXT, 
    	MIME TEXT
	);`

	if _, err = DB.Exec(query); err != nil {
		return err
	}

	query = `CREATE TABLE IF NOT EXISTS BOOKMARKS (
    	ID INTEGER PRIMARY KEY, 
    	SEC INTEGER, 
    	WATCH_COUNT INTEGER
	);`

	if _, err = DB.Exec(query); err != nil {
		return err
	}

	query = `CREATE TABLE IF NOT EXISTS SETTINGS (
    	KEY TEXT NOT NULL, 
		VALUE TEXT
	);`

	if _, err = DB.Exec(query); err != nil {
		return err
	}

	var count int
	if err = sq.Select("count(*)").From("OBJECTS").RunWith(DB).Scan(&count); err != nil {
		return err
	}

	if count == 0 {

		if err = makeGeneralFolder("root", "0", "-1"); err != nil {
			return err
		}

		if err = makeGeneralFolder("Music", MusicID, "0"); err != nil {
			return err
		}

		if err = makeGeneralFolder("Video", VideoID, "0"); err != nil {
			return err
		}

		if err = makeGeneralFolder("Images", ImageID, "0"); err != nil {
			return err
		}
	}

	return nil
}

func makeGeneralFolder(name string, objectId string, parentId string) error {
	obj := &Object{
		ObjectID: objectId,
		Class:    ClassFolder,
		ParentID: parentId,
		Name:     name,
	}
	return obj.Save()
}
