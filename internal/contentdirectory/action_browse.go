package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/db"
	"github.com/szonov/godlna/internal/dlna"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/internal/upnpav"
	"net/http"
	"path/filepath"
	"time"
)

type argInBrowse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int64
	RequestedCount int64
	SortCriteria   string
}

type argOutBrowse struct {
	Result         *soap.DIDLLite
	NumberReturned int
	TotalMatches   uint64
	UpdateID       string
}

func actionBrowse(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	in := &argInBrowse{}
	if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
		soap.SendError(err, w)
		return
	}

	object := db.GetObject(in.ObjectID, true)
	if object == nil {
		soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
		return
	}

	out := &argOutBrowse{}
	var objects []*db.Object

	switch in.BrowseFlag {
	case "BrowseDirectChildren":
		out.TotalMatches = object.ChildCount()
		objects = object.Children(in.RequestedCount, in.StartingIndex)

	case "BrowseMetadata":
		objects, out.TotalMatches = []*db.Object{object}, 1

	default:
		err := fmt.Errorf("invalid BrowseFlag: %s", in.BrowseFlag)
		soap.SendUPnPError(soap.ArgumentValueInvalidErrorCode, err.Error(), w)
		return
	}

	out.NumberReturned = len(objects)
	out.UpdateID = systemUpdateId
	out.Result = &soap.DIDLLite{
		Debug: r.Header.Get("X-Debug") == "1",
	}

	for _, o := range objects {
		switch o.Type {
		case db.TypeFolder:
			out.Result.Append(storageFolder(o))
		case db.TypeVideo:
			out.Result.Append(videoItem(o, r))
		case db.TypeStream:
			out.Result.Append(videoStream(o, r))
		}
	}

	soap.SendActionResponse(soapAction, out, w)
}

func protocolInfo(mimeType, contentFeatures string) string {
	return fmt.Sprintf("http-get:*:%s:%s", mimeType, contentFeatures)
}

func storageFolder(o *db.Object) upnpav.Container {
	return upnpav.Container{
		Object: upnpav.Object{
			ID:         o.ObjectID,
			Restricted: 1,
			ParentID:   o.ParentID,
			Class:      "object.container.storageFolder",
			Title:      o.Title(),
		},
	}
}

func videoItem(o *db.Object, r *http.Request) upnpav.Item {

	thumbURL := fmt.Sprintf("http://%s/v/t/%s/thumb.jpg", r.Host, o.ObjectID)
	videoURL := fmt.Sprintf("http://%s/v/v/%s/video%s", r.Host, o.ObjectID, filepath.Ext(o.Path))

	// bookmark
	var bookmark upnpav.Bookmark
	if dlna.UseSecondsInBookmark(r) {
		bookmark = upnpav.Bookmark(o.Bookmark.Duration().Seconds())
	} else {
		bookmark = upnpav.Bookmark(o.Bookmark.Duration().Milliseconds())
	}

	meta := o.Meta.(*db.VideoMeta)
	return upnpav.Item{
		Object: upnpav.Object{
			ID:          o.ObjectID,
			Restricted:  1,
			ParentID:    o.ParentID,
			Class:       "object.item.videoItem",
			Title:       o.Title(),
			Date:        o.Timestamp.Time().Format("2006-01-02T15:04:05"),
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},
		Bookmark: bookmark,
		Res: []upnpav.Resource{
			{
				URL:             videoURL,
				ProtocolInfo:    protocolInfo(o.MimeType(), dlna.NewMediaContentFeatures(o.Profile()).String()),
				Bitrate:         meta.BitRate,
				SampleFrequency: meta.SampleRate,
				Duration:        meta.Duration.String(),
				Size:            o.Size,
				Resolution:      meta.Resolution,
				AudioChannels:   meta.Channels,
			},
			{
				URL:          thumbURL,
				ProtocolInfo: protocolInfo("image/jpeg", dlna.NewThumbContentFeatures().String()),
			},
		},
	}
}

func videoStream(o *db.Object, r *http.Request) upnpav.Item {
	thumbURL := fmt.Sprintf("http://%s/s/t/%s/icon.png", r.Host, o.ObjectID)
	videoURL := fmt.Sprintf("http://%s/s/v/%s/stream.mp4", r.Host, o.ObjectID)

	return upnpav.Item{
		Object: upnpav.Object{
			ID:          o.ObjectID,
			Restricted:  1,
			ParentID:    o.ParentID,
			Class:       "object.item.videoItem",
			Title:       o.Title(),
			Date:        time.Now().Format("2006-01-02T15:04:05"),
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},
		Res: []upnpav.Resource{
			{
				URL:          videoURL,
				ProtocolInfo: protocolInfo(o.MimeType(), dlna.NewStreamContentFeatures(o.Profile()).String()),
			},
			{
				URL:          thumbURL,
				ProtocolInfo: protocolInfo("image/png", dlna.NewThumbContentFeatures().String()),
			},
		},
	}
}
