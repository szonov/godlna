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
	UpdateID       uint64
}

func actionBrowse(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	in := &argInBrowse{}
	out := &argOutBrowse{}

	if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
		soap.SendError(err, w)
		return
	}

	profile := client.GetProfileByRequest(r)
	var objects []*backend.Object

	switch in.BrowseFlag {
	case "BrowseDirectChildren":
		objects, out.TotalMatches = backend.GetObjectChildren(in.ObjectID, in.RequestedCount, in.StartingIndex)

	case "BrowseMetadata":
		object := backend.GetObject(in.ObjectID)
		if object == nil {
			soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
			return
		}
		objects = []*backend.Object{object}
		out.TotalMatches = 1

	default:
		err := fmt.Errorf("invalid BrowseFlag: %s", in.BrowseFlag)
		soap.SendUPnPError(soap.ArgumentValueInvalidErrorCode, err.Error(), w)
		return
	}

	out.NumberReturned = len(objects)
	out.UpdateID = serviceState.GetUint64("SystemUpdateID")
	out.Result = &soap.DIDLLite{
		Debug: strings.Contains(r.UserAgent(), "DIDLDebug"),
	}

	for _, o := range objects {
		if o.Type == backend.Video {
			out.Result.Append(videoItem(o, profile))
		} else {
			out.Result.Append(storageFolder(o))
		}
	}

	soap.SendActionResponse(soapAction, out, w)
}

func contentVideoFeatures() string {
	return "DLNA.ORG_OP=01;DLNA.ORG_FLAGS=01700000000000000000000000000000"
	//return "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000"
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

func videoItem(o *backend.Object, profile *client.Profile) upnpav.Item {

	// generate URLs for thumbnail and video
	thumbURL := "http://" + profile.Host + "/content/" + profile.Name + "/thumb/" + o.ObjectID + ".jpg"
	videoURL := "http://" + profile.Host + "/content/" + profile.Name + "/video/" + o.ObjectID + filepath.Ext(o.Path)

	// bookmark
	var dcmInfo string
	if o.Bookmark != nil && o.Bookmark.Uint64() > 0 {
		dcmInfo = fmt.Sprintf("BM=%d", profile.BookmarkResponseValue(o.Bookmark.Uint64()))
	}

	return upnpav.Item{
		Object: upnpav.Object{
			ID:         o.ObjectID,
			Restricted: 1,
			ParentID:   o.ParentID,
			Class:      "object.item.videoItem",
			Title:      o.Title(),
			Date:       o.Timestamp.Time().Format("2006-01-02T15:04:05"),
			// check - maybe it does not needed for TVs
			//Icon:        thumbURL,
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},
		DcmInfo: dcmInfo,
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
