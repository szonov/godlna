package contentdirectory

import (
	"encoding/json"
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/upnpav"
	"gopkg.in/vansante/go-ffprobe.v2"
	"path/filepath"
	"strconv"
	"time"
)

func transformObject(item *backend.Object, profile *client.Profile) (ret interface{}, err error) {

	objectID := item.ObjectID
	parentID := item.ParentID

	//if profile.UseVideoAsRoot() {
	//	switch backend.VideoID {
	//	case objectID:
	//		objectID = "0"
	//		parentID = "-1"
	//	case parentID:
	//		parentID = "0"
	//	}
	//}

	obj := upnpav.Object{
		ID:         objectID,
		Restricted: 1,
		ParentID:   parentID,
		Class:      "object." + item.Class,
		Title:      item.Title,
	}
	if item.Class == backend.ClassFolder {
		ret = upnpav.Container{
			Object:     obj,
			ChildCount: item.ChildrenCount,
		}
		return
	}

	obj.Date = time.Unix(item.Timestamp, 0).Format("2006-01-02T15:04:05")

	var meta *ffprobe.ProbeData
	if err = json.Unmarshal([]byte(item.MetaData), &meta); err != nil {
		return
	}

	if meta == nil {
		err = fmt.Errorf("no meta for '%s'", item.ObjectID)
		return
	}
	// test

	obj.Icon = "http://" + profile.Host + "/thumbs/" + profile.Name + "/" + item.ObjectID + ".jpg"
	obj.AlbumArtURI = &upnpav.AlbumArtURI{
		Value:   obj.Icon,
		Profile: "JPEG_TN",
	}

	res := make([]upnpav.Resource, 0)
	var size uint64
	size, err = strconv.ParseUint(meta.Format.Size, 10, 64)
	if err != nil {
		return
	}
	vstream := meta.FirstVideoStream()
	astream := meta.FirstAudioStream()
	res = append(res, upnpav.Resource{
		URL: "http://" + profile.Host + "/video/" + profile.Name + "/" + item.ObjectID + filepath.Ext(item.Path),
		//ProtocolInfo: "http-get:*:video/avi:DLNA.ORG_OP=01;DLNA.ORG_FLAGS=01700000000000000000000000000000",
		ProtocolInfo: "http-get:*:video/x-msvideo:DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000",
		//ProtocolInfo:    "http-get:*:video/x-mkv:DLNA.ORG_PN=MATROSKA;DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=21D00000000000000000000000000000",
		Bitrate:         backend.FmtBitrate(meta.Format.BitRate),
		SampleFrequency: astream.SampleRate,
		Duration:        backend.FmtDuration(meta.Format.Duration()),
		Size:            size,
		Resolution:      fmt.Sprintf("%dx%d", vstream.Width, vstream.Height),
		AudioChannels:   strconv.Itoa(astream.Channels),
	})

	// icon
	res = append(res, upnpav.Resource{
		URL:          obj.Icon,
		ProtocolInfo: "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN;DLNA.ORG_FLAGS=00f00000000000000000000000000000",
	})

	dcmInfo := ""
	if item.Bookmark > 0 {
		dcmInfo = fmt.Sprintf("BM=%d", profile.BookmarkResponseValue(item.Bookmark))
	}

	ret = upnpav.Item{
		Object:  obj,
		DcmInfo: dcmInfo,
		Res:     res,
	}
	return
}
