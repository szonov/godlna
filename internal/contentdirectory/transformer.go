package contentdirectory

import (
	"encoding/json"
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/upnpav"
	"gopkg.in/vansante/go-ffprobe.v2"
	"path/filepath"
)

func transformContainer(o *backend.Object) upnpav.Container {
	return upnpav.Container{
		Object: upnpav.Object{
			ID:         o.ObjectID,
			Restricted: 1,
			ParentID:   o.ParentID,
			Class:      "object.container.storageFolder",
			Title:      o.Title,
		},
		ChildCount: o.Size.Uint64(),
	}
}

func transformVideo(o *backend.Object, profile *client.Profile) (ret upnpav.Item, err error) {

	var meta *ffprobe.ProbeData
	if err = json.Unmarshal([]byte(o.MetaData), &meta); err != nil {
		return
	}

	if meta == nil {
		err = fmt.Errorf("no meta for '%s'", o.ObjectID)
		return
	}

	// generate URLs for thumbnail and video
	thumbURL := "http://" + profile.Host + "/content/" + profile.Name + "/thumb/" + o.ObjectID + ".jpg"
	videoURL := "http://" + profile.Host + "/content/" + profile.Name + "/video/" + o.ObjectID + filepath.Ext(o.Path)

	// bookmark
	var dcmInfo string
	if o.Bookmark != nil && o.Bookmark.Uint64() > 0 {
		dcmInfo = fmt.Sprintf("BM=%d", profile.BookmarkResponseValue(o.Bookmark.Uint64()))
	}

	ret = upnpav.Item{
		Object: upnpav.Object{
			ID:          o.ObjectID,
			Restricted:  1,
			ParentID:    o.ParentID,
			Class:       "object.item.videoItem",
			Title:       o.Title,
			Date:        o.Timestamp.Time().Format("2006-01-02T15:04:05"),
			Icon:        thumbURL,
			AlbumArtURI: &upnpav.AlbumArtURI{Value: thumbURL, Profile: "JPEG_TN"},
		},
		DcmInfo: dcmInfo,
		Res: []upnpav.Resource{
			{
				URL:             videoURL,
				ProtocolInfo:    fmt.Sprintf("http-get:*:%s:%s", o.MimeType, contentFeatures()),
				Bitrate:         o.BitRate.Uint(),
				SampleFrequency: o.SampleRate.String(),
				Duration:        o.DurationSec.String(),
				Size:            o.Size.Uint64(),
				Resolution:      o.Resolution.String(),
				AudioChannels:   o.Channels.Int(),
			},
			{
				URL:          thumbURL,
				ProtocolInfo: "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN;DLNA.ORG_FLAGS=00f00000000000000000000000000000",
			},
		},
	}
	return
}

func transformObject(o *backend.Object, profile *client.Profile) (interface{}, error) {

	if o.Type == backend.Folder {
		return transformContainer(o), nil
	}

	if o.Type == backend.Video {
		return transformVideo(o, profile)
	}

	return nil, fmt.Errorf("unknown type '%d' for object '%s'", o.Type, o.ObjectID)
}

func contentFeatures() string {
	return "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000"
}
