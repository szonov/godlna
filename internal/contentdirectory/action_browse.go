package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/internal/store"
	"github.com/szonov/godlna/internal/upnpav"
	"net/http"
	"path/filepath"
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

	object := store.GetObject(in.ObjectID, true)
	if object == nil {
		soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
		return
	}

	out := &argOutBrowse{}
	var objects []*store.Object

	switch in.BrowseFlag {
	case "BrowseDirectChildren":
		out.TotalMatches = object.ChildCount()
		objects = object.Children(in.RequestedCount, in.StartingIndex)

	case "BrowseMetadata":
		objects, out.TotalMatches = []*store.Object{object}, 1

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

	features := client.GetFeatures(r)
	for _, o := range objects {
		if o.IsFolder() {
			out.Result.Append(storageFolder(o))
		} else {
			out.Result.Append(videoItem(o, r.Host, features))
		}
	}

	soap.SendActionResponse(soapAction, out, w)
}

func contentVideoFeatures() string {
	return "DLNA.ORG_OP=01;DLNA.ORG_FLAGS=01700000000000000000000000000000"
}

func contentThumbnailFeatures() string {
	return "DLNA.ORG_PN=JPEG_TN;DLNA.ORG_FLAGS=00f00000000000000000000000000000"
}

func protocolInfo(mimeType, contentFeatures string) string {
	return fmt.Sprintf("http-get:*:%s:%s", mimeType, contentFeatures)
}

func storageFolder(o *store.Object) upnpav.Container {
	return upnpav.Container{
		Object: upnpav.Object{
			ID:         o.ObjectID,
			Restricted: 1,
			ParentID:   o.ParentID,
			Class:      "object." + o.Class,
			Title:      o.Title(),
		},
	}
}

func videoItem(o *store.Object, host string, features *client.Features) upnpav.Item {

	bm := o.Bookmark.Duration().String()

	// thumbnail type (normal/square)
	thumbType := "n"
	if features.UseSquareThumbnails {
		thumbType = "s"
	}

	// URLs for thumbnail and video
	thumbURL := fmt.Sprintf("http://%s/t/%s/%s/%s.jpg", host, bm, thumbType, o.ObjectID)
	videoURL := fmt.Sprintf("http://%s/v/%s/%s%s", host, bm, o.ObjectID, filepath.Ext(o.Path))

	// bookmark
	var bookmark upnpav.Bookmark
	if features.UseSecondsInBookmark {
		bookmark = upnpav.Bookmark(o.Bookmark.Duration().Seconds())

	} else {
		bookmark = upnpav.Bookmark(o.Bookmark.Duration().Milliseconds())
	}

	return upnpav.Item{
		Object: upnpav.Object{
			ID:          o.ObjectID,
			Restricted:  1,
			ParentID:    o.ParentID,
			Class:       "object." + o.Class,
			Title:       o.Title(),
			Date:        o.Timestamp.Time().Format("2006-01-02T15:04:05"),
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},

		Bookmark: bookmark,

		Res: []upnpav.Resource{
			{
				URL:             videoURL,
				ProtocolInfo:    protocolInfo(o.MimeType(), contentVideoFeatures()),
				Bitrate:         o.BitRate.Uint(),
				SampleFrequency: o.SampleRate.String(),
				Duration:        o.Duration.String(),
				Size:            o.Size.Uint64(),
				Resolution:      o.Resolution.String(),
				AudioChannels:   o.Channels.Int(),
			},
			{
				URL:          thumbURL,
				ProtocolInfo: protocolInfo("image/jpeg", contentThumbnailFeatures()),
			},
		},
	}
}
