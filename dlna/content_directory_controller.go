package dlna

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/szonov/godlna/indexer"
	"github.com/szonov/godlna/soap"
	"github.com/szonov/godlna/upnp/events"
	"github.com/szonov/godlna/upnpav"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type (
	ContentDirectoryController struct {
		serviceDescriptionXML []byte
		eventManager          *events.Manager
		systemUpdateId        string
		srv                   *Server
	}
	argInBrowse struct {
		ObjectID       int64
		BrowseFlag     string
		Filter         string
		StartingIndex  int64
		RequestedCount int64
		SortCriteria   string
	}

	argOutBrowse struct {
		Result         *soap.DIDLLite
		NumberReturned int
		TotalMatches   int64
		UpdateID       string
	}
	argOutGetFeatureList struct {
		FeatureList soap.XMLLite
	}
	argInSetBookmark struct {
		CategoryType string
		RID          string
		ObjectID     int64
		PosSecond    int64
	}
	dbObject struct {
		ID         int64
		Path       string
		Typ        int
		Format     string
		FileSize   int64
		VideoCodec string
		AudioCodec string
		Width      int
		Height     int
		Channels   int
		Bitrate    int
		Frequency  int
		Duration   int64
		Bookmark   sql.NullInt64
		Date       int64
	}
)

func NewContentDirectoryController(srv *Server) (*ContentDirectoryController, error) {
	var err error
	ctl := &ContentDirectoryController{
		eventManager:   events.NewManager(),
		systemUpdateId: "1",
		srv:            srv,
	}

	if ctl.serviceDescriptionXML, err = xml.Marshal(makeContentDirectoryServiceDescription()); err != nil {
		return ctl, err
	}
	ctl.serviceDescriptionXML = append([]byte(xml.Header), ctl.serviceDescriptionXML...)

	return ctl, nil
}

func (ctl *ContentDirectoryController) HandleSCPDURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXML(ctl.serviceDescriptionXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (ctl *ContentDirectoryController) HandleEventSubURL(w http.ResponseWriter, r *http.Request) {
	ctl.eventManager.HandleEventSubURL(w, r, func() map[string]string {
		return map[string]string{
			"SystemUpdateID":     ctl.systemUpdateId,
			"ContainerUpdateIDs": "",
		}
	})
}

func (ctl *ContentDirectoryController) HandleControlURL(w http.ResponseWriter, r *http.Request) {
	// Control URL works only with POST http method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// resolve current action name from http header
	soapAction := soap.DetectAction(r.Header.Get("SoapAction"))
	if soapAction == nil || soapAction.ServiceType != ContentDirectoryServiceType {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch soapAction.Name {
	case "Browse":
		in := &argInBrowse{}
		if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
			soap.SendError(err, w)
			return
		}

		obj := ctl.obj(in.ObjectID)
		if obj == nil {
			soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
			return
		}

		out := &argOutBrowse{
			Result: &soap.DIDLLite{
				Debug: r.Header.Get("X-Debug") == "1",
			},
			UpdateID: ctl.systemUpdateId,
		}

		switch in.BrowseFlag {
		case "BrowseDirectChildren":
			// TotalMatches
			q := "SELECT count(*) FROM objects where path like $1 AND path NOT LIKE $2"
			_ = ctl.srv.Psql.QueryRow(context.Background(), q, obj.Path+"/%", obj.Path+"/%/%").Scan(&out.TotalMatches)

			// Result
			q = "SELECT " + ctl.fields() + " FROM objects where path like $1 AND path NOT LIKE $2 ORDER BY typ, path LIMIT $3 OFFSET $4"
			rows, err := ctl.srv.Psql.Query(context.Background(), q, obj.Path+"/%", obj.Path+"/%/%", in.RequestedCount, in.StartingIndex)
			if err == nil {
				for rows.Next() {
					o := &dbObject{}
					if err = ctl.scan(rows, o); err != nil {
						slog.Error("unable to scan queue row", err.Error())
					} else {
						out.Result.Append(ctl.upnpavObj(o, obj.ID, r))
					}
				}
				rows.Close()
			}
		case "BrowseMetadata":
			var parentId int64
			if obj.ID == 0 {
				parentId = -1
			} else {
				parentPath := filepath.Dir(obj.Path)
				if parentPath == ctl.srv.RootDirectory {
					parentId = 0
				} else {
					q := "SELECT id FROM objects where path = $1"
					_ = ctl.srv.Psql.QueryRow(context.Background(), q, parentPath).Scan(&parentId)
				}
			}
			out.TotalMatches = 1
			out.Result.Append(ctl.upnpavObj(obj, parentId, r))

		default:
			err := fmt.Errorf("invalid BrowseFlag: %s", in.BrowseFlag)
			soap.SendUPnPError(soap.ArgumentValueInvalidErrorCode, err.Error(), w)
			return
		}

		out.NumberReturned = len(out.Result.Items)

		w.Header().Set("EXT", "")
		soap.SendActionResponse(soapAction, out, w)

	case "GetSearchCapabilities":
		w.Header().Set("EXT", "")
		soap.SendActionResponse(soapAction, "<SearchCaps></SearchCaps>", w)

	case "GetSortCapabilities":
		w.Header().Set("EXT", "")
		soap.SendActionResponse(soapAction, "<SortCaps></SortCaps>", w)

	case "GetSystemUpdateID":
		w.Header().Set("EXT", "")
		soap.SendActionResponse(soapAction, fmt.Sprintf("<Id>%s</Id>", ctl.systemUpdateId), w)

	case "X_GetFeatureList":
		out := &argOutGetFeatureList{
			FeatureList: `<Features xmlns="urn:schemas-upnp-org:av:avs"` +
				` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` +
				` xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">` +
				`<Feature name="samsung.com_BASICVIEW" version="1">` +
				`<container id="0" type="object.item.videoItem"/>` +
				`</Feature>` +
				`</Features>`,
		}
		w.Header().Set("EXT", "")
		soap.SendActionResponse(soapAction, out, w)

	case "X_SetBookmark":
		in := &argInSetBookmark{}
		if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
			soap.SendError(err, w)
			return
		}
		if useSecondsInBookmark(r) {
			in.PosSecond *= 1000
		}

		var fullPath string
		var duration int64
		var bookmark sql.NullInt64

		err := ctl.srv.Psql.QueryRow(context.Background(),
			"UPDATE objects SET bookmark = $1 WHERE id = $2 AND typ = $3 RETURNING path, duration, bookmark",
			in.PosSecond,
			in.ObjectID,
			indexer.TypeVideo,
		).Scan(&fullPath, &duration, &bookmark)

		if err != nil {
			slog.Error(err.Error())
			soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
			return
		}
		indexer.MakeDSMStyleThumbnail(fullPath, duration, bookmark, true)
		soap.SendActionResponse(soapAction, nil, w)

	default:
		err := fmt.Errorf("unknown action '%s'", soapAction.Name)
		soap.SendUPnPError(soap.InvalidActionErrorCode, err.Error(), w, http.StatusUnauthorized)
	}
}

func (ctl *ContentDirectoryController) HandleContentURL(w http.ResponseWriter, r *http.Request) {
	obj := r.PathValue("obj")
	ext := filepath.Ext(obj)
	oid := strings.TrimSuffix(obj, ext)
	objectID, err := strconv.ParseInt(oid, 10, 64)
	if err != nil {
		fmt.Println("Error converting string to int64:", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	o := ctl.obj(objectID)
	if o == nil {
		slog.Error("Object path not found", "objectID", objectID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("EXT", "")

	if ext == ".jpg" {
		w.Header().Set("transferMode.dlna.org", "Interactive")
		w.Header().Set("contentFeatures.dlna.org", o.thumbProtocolInfo())
		w.Header().Set("Content-Type", "image/jpeg")
		http.ServeFile(w, r, indexer.ThumbPath(o.Path))
		return
	}

	w.Header().Set("transferMode.dlna.org", "Streaming")
	w.Header().Set("contentFeatures.dlna.org", o.videoProtocolInfo())
	w.Header().Set("Content-Type", o.MimeType())
	http.ServeFile(w, r, o.Path)
}

func (ctl *ContentDirectoryController) obj(objectID int64) *dbObject {
	var err error
	var row pgx.Row
	ctx := context.Background()
	if objectID <= 0 {
		row = ctl.srv.Psql.QueryRow(ctx, "SELECT "+ctl.fields()+" FROM objects WHERE path = $1", ctl.srv.RootDirectory)
	} else {
		row = ctl.srv.Psql.QueryRow(ctx, "SELECT "+ctl.fields()+" FROM objects WHERE id = $1", objectID)
	}
	obj := &dbObject{}
	if err = ctl.scan(row, obj); err != nil {
		return nil
	}
	return obj
}

func (ctl *ContentDirectoryController) fields() string {
	return "id, path, typ, format, file_size, video_codec, audio_codec," +
		"width, height, channels, bitrate, frequency, duration, bookmark, date"
}

func (ctl *ContentDirectoryController) scan(row pgx.Row, o *dbObject) error {
	err := row.Scan(&o.ID, &o.Path, &o.Typ, &o.Format, &o.FileSize, &o.VideoCodec, &o.AudioCodec, &o.Width, &o.Height,
		&o.Channels, &o.Bitrate, &o.Frequency, &o.Duration, &o.Bookmark, &o.Date)
	if err != nil {
		return err
	}
	if o.Path == ctl.srv.RootDirectory {
		o.ID = 0
	}
	return nil
}

func (ctl *ContentDirectoryController) upnpavObj(o *dbObject, parentID int64, r *http.Request) any {
	if o.Typ == indexer.TypeFolder {
		return upnpav.Container{
			Object: upnpav.Object{
				ID:         strconv.FormatInt(o.ID, 10),
				Restricted: 1,
				ParentID:   strconv.FormatInt(parentID, 10),
				Class:      "object.container.storageFolder",
				Title:      filepath.Base(o.Path),
			},
		}
	}

	thumbURL := fmt.Sprintf("http://%s/ct/t/%d.jpg", r.Host, o.ID)
	videoURL := fmt.Sprintf("http://%s/ct/v/%d%s", r.Host, o.ID, filepath.Ext(o.Path))

	// bookmark
	var bookmark upnpav.Bookmark
	if useSecondsInBookmark(r) {
		bookmark = upnpav.Bookmark(o.Bookmark.Int64 / 1000)
	} else {
		bookmark = upnpav.Bookmark(o.Bookmark.Int64)
	}

	return upnpav.Item{
		Object: upnpav.Object{
			ID:          strconv.FormatInt(o.ID, 10),
			Restricted:  1,
			ParentID:    strconv.FormatInt(parentID, 10),
			Class:       "object.item.videoItem",
			Title:       indexer.NameWithoutExtension(filepath.Base(o.Path)),
			Date:        time.Unix(o.Date, 0).Format("2006-01-02T15:04:05"),
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},
		Bookmark: bookmark,
		Res: []upnpav.Resource{
			{
				URL:             videoURL,
				ProtocolInfo:    o.videoProtocolInfo(),
				Bitrate:         o.Bitrate,
				SampleFrequency: o.Frequency,
				Duration:        indexer.DurationString(o.Duration),
				Size:            o.FileSize,
				Resolution:      fmt.Sprintf("%dx%d", o.Width, o.Height),
				AudioChannels:   o.Channels,
			},
			{
				URL:          thumbURL,
				ProtocolInfo: o.thumbProtocolInfo(),
			},
		},
	}
}

func (o *dbObject) MimeType() string {
	if strings.Contains(o.Format, "matroska") || strings.Contains(o.Format, "avi") {
		return "video/avi"
	}
	return "video/x-msvideo"
}

func (o *dbObject) videoProtocolInfo() string {
	return fmt.Sprintf(
		"http-get:*:%s:DLNA.ORG_OP=01;DLNA.ORG_FLAGS=01700000000000000000000000000000",
		o.MimeType(),
	)
}
func (o *dbObject) thumbProtocolInfo() string {
	return "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN;DLNA.ORG_FLAGS=00f00000000000000000000000000000"
}
