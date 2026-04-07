package dlna

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/szonov/godlna/dlna/backend"
	"github.com/szonov/godlna/pkg/ffmpeg"
	"github.com/szonov/godlna/pkg/soap"
	"github.com/szonov/godlna/pkg/upnp/events"
	"github.com/szonov/godlna/pkg/upnpav"
)

type (
	ContentDirectoryController struct {
		serviceDescriptionXML []byte
		eventManager          *events.Manager
		systemUpdateId        string
		back                  *backend.Backend
	}
	argInBrowse struct {
		ObjectID       int
		BrowseFlag     string
		Filter         string
		StartingIndex  int
		RequestedCount int
		SortCriteria   string
	}

	argOutBrowse struct {
		Result         *soap.DIDLLite
		NumberReturned int
		TotalMatches   int
		UpdateID       string
	}
	argOutGetFeatureList struct {
		FeatureList soap.XMLLite
	}
	argInSetBookmark struct {
		CategoryType string
		RID          string
		ObjectID     int
		PosSecond    int64
	}
)

func NewContentDirectoryController(back *backend.Backend) (*ContentDirectoryController, error) {
	var err error
	ctl := &ContentDirectoryController{
		eventManager:   events.NewManager(),
		systemUpdateId: "1",
		back:           back,
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

		o, err := ctl.back.Object(in.ObjectID)
		if err != nil {
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
			children, err := ctl.back.Children(o, in.RequestedCount, in.StartingIndex)
			if err != nil {
				soap.SendError(err, w)
				return
			}
			out.TotalMatches = children.TotalMatches
			for _, child := range children.Items {
				out.Result.Append(ctl.upnpavObj(child, o.ID, r))
			}
		case "BrowseMetadata":
			parentId, err := ctl.back.ParentId(o)
			if err != nil {
				soap.SendError(err, w)
				return
			}
			out.TotalMatches = 1
			out.Result.Append(ctl.upnpavObj(o, parentId, r))

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

		if err := ctl.back.SetBookmark(in.ObjectID, in.PosSecond); err != nil {
			soap.SendError(err, w)
		}
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

	objectID, err := strconv.Atoi(oid)
	if err != nil {
		fmt.Println("Error converting string to int:", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	o, err := ctl.back.Object(objectID)
	if err != nil {
		slog.Error("Object not found", "objectID", objectID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("EXT", "")

	if ext == ".jpg" {
		w.Header().Set("transferMode.dlna.org", "Interactive")
		w.Header().Set("contentFeatures.dlna.org", ctl.thumbProtocolInfo(o))
		w.Header().Set("Content-Type", "image/jpeg")
		http.ServeFile(w, r, o.ThumbPath())
		return
	}

	w.Header().Set("transferMode.dlna.org", "Streaming")
	w.Header().Set("contentFeatures.dlna.org", ctl.videoProtocolInfo(o))
	w.Header().Set("Content-Type", ctl.mimeType(o))
	http.ServeFile(w, r, o.Path)
}

func (ctl *ContentDirectoryController) upnpavObj(o *backend.Object, parentID int, r *http.Request) any {
	if o.Typ == backend.ObjectFolder {
		return upnpav.Container{
			Object: upnpav.Object{
				ID:         strconv.Itoa(o.ID),
				Restricted: 1,
				ParentID:   strconv.Itoa(parentID),
				Class:      "object.container.storageFolder",
				Title:      o.Title(),
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
			ID:          strconv.Itoa(o.ID),
			Restricted:  1,
			ParentID:    strconv.Itoa(parentID),
			Class:       "object.item.videoItem",
			Title:       o.Title(),
			Date:        time.Unix(o.Date, 0).Format("2006-01-02T15:04:05"),
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},
		Bookmark: bookmark,
		Res: []upnpav.Resource{
			{
				URL:             videoURL,
				ProtocolInfo:    ctl.videoProtocolInfo(o),
				Bitrate:         o.Bitrate,
				SampleFrequency: o.Frequency,
				Duration:        ffmpeg.DurationToString(time.Duration(o.Duration) * time.Millisecond),
				Size:            o.FileSize,
				Resolution:      fmt.Sprintf("%dx%d", o.Width, o.Height),
				AudioChannels:   o.Channels,
			},
			{
				URL:          thumbURL,
				ProtocolInfo: ctl.thumbProtocolInfo(o),
			},
		},
	}
}

func (ctl *ContentDirectoryController) mimeType(o *backend.Object) string {
	if strings.Contains(o.Format, "matroska") || strings.Contains(o.Format, "avi") {
		return "video/avi"
	}
	return "video/x-msvideo"
}

func (ctl *ContentDirectoryController) videoProtocolInfo(o *backend.Object) string {
	return fmt.Sprintf(
		"http-get:*:%s:DLNA.ORG_OP=01;DLNA.ORG_FLAGS=01700000000000000000000000000000",
		ctl.mimeType(o),
	)
}

func (ctl *ContentDirectoryController) thumbProtocolInfo(o *backend.Object) string {
	return "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN;DLNA.ORG_FLAGS=00f00000000000000000000000000000"
}
