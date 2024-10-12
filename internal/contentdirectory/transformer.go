package contentdirectory

import (
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/upnpav"
	"net/url"
	"time"
)

func transformObject(item *backend.Object, profile *client.Profile) (ret interface{}, err error) {
	objectID := item.ObjectID
	parentID := item.ParentID
	if profile.UseVideoAsRoot() {
		switch backend.VideoID {
		case objectID:
			objectID = "0"
			parentID = "-1"
		case parentID:
			parentID = "0"
		}
	}

	obj := upnpav.Object{
		ID:         objectID,
		Restricted: 1,
		ParentID:   parentID,
		Class:      "object." + item.Class,
		Title:      item.Title,
		Date: upnpav.Timestamp{
			Time: time.Unix(item.Timestamp, 0),
		},
	}
	if item.Class == backend.ClassFolder {
		ret = upnpav.Container{
			Object:     obj,
			ChildCount: item.ChildrenCount,
		}
		return
	}

	//iconURI := (&url.URL{
	//	Scheme: "http",
	//	Host:   host,
	//	Path:   iconPath,
	//	RawQuery: url.Values{
	//		"path": {cdsObject.Path},
	//	}.Encode(),
	//}).String()

	iconURI := &url.URL{
		Scheme: "http",
		Host:   profile.Host,
		Path:   "/thumbs/" + profile.Name + "/" + item.ObjectID + ".jpg",
	}
	obj.Icon = iconURI.String()
	ret = upnpav.Item{
		Object: obj,
	}
	return
}
