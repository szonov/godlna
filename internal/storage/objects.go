package storage

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"log/slog"
	"strconv"
	"strings"
)

const (
	ClassFolder    = "container.storageFolder"
	ClassItemVideo = "item.videoItem"
)

type (
	Object struct {
		ID       int64  `json:"ID"`
		ObjectID string `json:"OBJECT_ID"`
		ParentID string `json:"PARENT_ID"`
		//RefID    string   `json:"REF_ID,omitempty"`
		Class    string   `json:"CLASS"`
		DetailID int64    `json:"DETAIL_ID,omitempty"`
		Name     string   `json:"NAME,omitempty"`
		Details  *Details `json:"-"`
	}

	Details struct {
		ID         int64  `json:"ID"`
		Path       string `json:"PATH,omitempty"`
		Size       int64  `json:"SIZE,omitempty"`
		Timestamp  int64  `json:"TIMESTAMP,omitempty"`
		Title      string `json:"TITLE"`
		Duration   string `json:"DURATION,omitempty"`
		Bitrate    int64  `json:"BITRATE,omitempty"`
		SampleRate int64  `json:"SAMPLERATE,omitempty"`
		Creator    string `json:"CREATOR,omitempty"`
		Artist     string `json:"ARTIST,omitempty"`
		Album      string `json:"ALBUM,omitempty"`
		Genre      string `json:"GENRE,omitempty"`
		Comment    string `json:"COMMENT,omitempty"`
		Channels   int    `json:"CHANNELS,omitempty"`
		Track      int    `json:"TRACK,omitempty"`
		Resolution string `json:"RESOLUTION,omitempty"`
		Thumbnail  bool   `json:"THUMBNAIL,omitempty"`
		AlbumArt   bool   `json:"ALBUM_ART,omitempty"`
		Rotation   int    `json:"ROTATION,omitempty"`
		DlnaPN     string `json:"DLNA_PN,omitempty"`
		Mime       string `json:"MIME,omitempty"`
	}
)

func (o *Object) Save(skipDetailsSave ...bool) (err error) {
	if o.ParentID == "" {
		err = fmt.Errorf("undefined ParentId")
		return
	}

	if o.ObjectID == "" {
		o.ObjectID = getNewObjectId(o.ParentID)
	}

	withDetails := true
	if len(skipDetailsSave) > 0 && skipDetailsSave[0] {
		withDetails = false
	}

	if withDetails {
		// if details is not set, create new one with default parameters
		if o.Details == nil {
			o.Details = &Details{
				ID:    o.DetailID,
				Title: o.Name,
			}
		}
		// this is a programmers error -> fail
		if o.DetailID != o.Details.ID {
			return fmt.Errorf("object '%s' has different detail id o.DetailID = '%s', o.Details.ID = '%s'",
				o.ObjectID, o.DetailID, o.Details.ID)
		}
		// synchronize name of object and title in details if one have title and other have no
		if o.Details.Title == "" && o.Name != "" {
			o.Details.Title = o.Name
		} else if o.Name == "" && o.Details.Title != "" {
			o.Name = o.Details.Title
		}
		// save details
		if err = o.Details.Save(); err != nil {
			return fmt.Errorf("save object details '%s': %w", o.ObjectID, err)
		}
		// update objects Detail ID
		o.DetailID = o.Details.ID
	}

	// case when saving without details, but know details and objects name is empty
	if o.Name == "" && o.Details != nil {
		o.Name = o.Details.Title
	}

	storeData := map[string]any{
		"OBJECT_ID": o.ObjectID,
		"PARENT_ID": o.ParentID,
		"CLASS":     o.Class,
		"NAME":      o.Name,
	}
	if o.Details.ID > 0 {
		storeData["DETAIL_ID"] = o.Details.ID
	}

	slog.Debug("Save Object", "obj", storeData, "ID", o.ID)

	var res sql.Result

	// update
	if o.ID != 0 {
		_, err = sq.Update("OBJECTS").SetMap(storeData).Where("ID = ?", o.ID).RunWith(DB).Exec()
		return
	}
	// insert
	if res, err = sq.Insert("OBJECTS").SetMap(storeData).RunWith(DB).Exec(); err != nil {
		return
	}

	o.ID, err = res.LastInsertId()
	return
}

func (d *Details) Save() (err error) {

	storeData := map[string]any{
		"TITLE": d.Title,
	}
	if d.Path != "" {
		storeData["PATH"] = d.Path
	}
	if d.Size > 0 {
		storeData["SIZE"] = d.Size
	}
	if d.Timestamp > 0 {
		storeData["TIMESTAMP"] = d.Timestamp
	}
	if d.Duration != "" {
		storeData["DURATION"] = d.Duration
	}
	if d.Bitrate > 0 {
		storeData["BITRATE"] = d.Bitrate
	}
	if d.SampleRate > 0 {
		storeData["SAMPLERATE"] = d.SampleRate
	}

	slog.Debug("Save Details", "det", storeData, "ID", d.ID)

	var res sql.Result
	// update
	if d.ID != 0 {
		_, err = sq.Update("DETAILS").SetMap(storeData).Where("ID = ?", d.ID).RunWith(DB).Exec()
		return
	}

	// insert
	if res, err = sq.Insert("DETAILS").SetMap(storeData).RunWith(DB).Exec(); err != nil {
		return
	}
	d.ID, err = res.LastInsertId()
	return
}

func getNextAvailableId(parentID string) int64 {
	var err error
	var maxObjectID string
	query := `SELECT OBJECT_ID from OBJECTS where ID = (SELECT max(ID) from OBJECTS where PARENT_ID = ?)`
	row := DB.QueryRow(query, parentID)
	err = row.Scan(&maxObjectID)
	if err == nil {
		if p := strings.LastIndex(maxObjectID, "$"); p != -1 {
			var maxValue int64
			if maxValue, err = strconv.ParseInt(maxObjectID[p+1:], 10, 64); err == nil {
				return maxValue + 1
			}
		}
	}
	return 0
}

func getNewObjectId(parentID string) string {
	return parentID + "$" + strconv.FormatInt(getNextAvailableId(parentID), 10)
}
