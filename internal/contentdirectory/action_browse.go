package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/internal/upnpav"
	"net/http"
	"path/filepath"
	"strings"
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

	object := backend.GetObject(in.ObjectID)
	if object == nil {
		soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
		return
	}

	out := &argOutBrowse{}
	profile := client.GetProfileByRequest(r)
	var objects []*backend.Object

	switch in.BrowseFlag {
	case "BrowseDirectChildren":
		objects, out.TotalMatches = object.Children(in.RequestedCount, in.StartingIndex)

	case "BrowseMetadata":
		objects, out.TotalMatches = []*backend.Object{object}, 1

	default:
		err := fmt.Errorf("invalid BrowseFlag: %s", in.BrowseFlag)
		soap.SendUPnPError(soap.ArgumentValueInvalidErrorCode, err.Error(), w)
		return
	}

	out.NumberReturned = len(objects)
	out.UpdateID = backend.GetSystemUpdateId()
	out.Result = &soap.DIDLLite{
		Debug: strings.Contains(r.UserAgent(), "DIDLDebug"),
	}

	for _, o := range objects {
		if o.Type == backend.Video {
			out.Result.Append(videoItem(o, r.Host, profile))
		} else {
			out.Result.Append(storageFolder(o))
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

func storageFolder(o *backend.Object) upnpav.Container {
	return upnpav.Container{
		Object: upnpav.Object{
			ID:         o.ObjectID,
			Restricted: 1,
			ParentID:   o.ParentID,
			Class:      "object.container.storageFolder",
			Title:      o.Title(),
		},
		ChildCount: o.Size.Uint64(),
	}
}

func videoItem(o *backend.Object, host string, profile *client.Profile) upnpav.Item {

	// thumbnail type (normal/square)
	thumbType := "n"
	if profile.UseSquareThumbnails() {
		thumbType = "s"
	}

	// URLs for thumbnail and video
	thumbURL := fmt.Sprintf("http://%s/t/%s/%s.jpg", host, thumbType, o.ObjectID)
	videoURL := fmt.Sprintf("http://%s/v/%s%s", host, o.ObjectID, filepath.Ext(o.Path))

	// bookmark
	var bookmark int64
	if profile.UseBookmarkMilliseconds() {
		bookmark = o.Bookmark.Duration().Milliseconds()
	} else {
		bookmark = int64(o.Bookmark.Duration().Seconds())
	}

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

		Bookmark: upnpav.Bookmark(bookmark),

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
