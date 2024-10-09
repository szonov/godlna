package contentdirectory1

import (
	"github.com/szonov/godlna/soap"
	"net/http"
	"time"
)

type argInBrowse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  uint32
	RequestedCount uint32
	SortCriteria   string
}

type argOutBrowse struct {
	Result         string
	NumberReturned uint32
	TotalMatches   uint32
	UpdateID       uint32
	SomeString     string
}

func (ctl *Controller) actionBrowse(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	// input
	in := &argInBrowse{}
	if err := soap.UnmarshalEnvelopeBody(r.Body, in); err != nil {
		ctl.error(err, w)
		return
	}

	// output
	out := &argOutBrowse{}

	out.NumberReturned = 4
	out.TotalMatches = 4
	out.UpdateID = ctl.state.GetUint32("SystemUpdateID")
	out.Result = `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/"
	          xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/"
	          xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">
	   <container id="64" parentID="0" restricted="1" searchable="1" childCount="1">
	       <dc:title>Browse Folders
	       </dc:title>
	       <upnp:class>object.container.storageFolder</upnp:class>
	   </container>
	   <container id="1" parentID="0" restricted="1" searchable="1" childCount="7">
	       <dc:title>Music</dc:title>
	       <upnp:class>object.container.storageFolder</upnp:class>
	   </container>
	   <container id="3" parentID="0" restricted="1" searchable="1" childCount="5">
	       <dc:title>Pictures</dc:title>
	       <upnp:class>object.container.storageFolder</upnp:class>
	   </container>
	   <container id="2" parentID="0" restricted="1" searchable="1" childCount="3">
	       <dc:title>Video</dc:title>
	       <upnp:class>object.container.storageFolder</upnp:class>
	   </container>
	</DIDL-Lite>`

	out.SomeString = ctl.state.Get("SystemUpdateID")

	go func() {
		time.Sleep(5 * time.Second)
		v := ctl.state.GetUint32("SystemUpdateID") + 1
		ctl.state.SetUint32("SystemUpdateID", v)
		ctl.state.NotifyChanges("SystemUpdateID")
	}()

	soapAction.WithResponse(out).Send(w)
}
