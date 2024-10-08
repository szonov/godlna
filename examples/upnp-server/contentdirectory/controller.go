package contentdirectory

import (
	_ "embed"
	"github.com/szonov/go-upnp-lib"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

//go:embed scpd_template.xml
var embedServiceXML []byte

func NewServiceController() *upnp.ServiceController {
	h := new(MyHandler)

	ctl := upnp.NewServiceController(
		&upnp.Service{
			ServiceType: ServiceType,
			ServiceId:   ServiceId,
			SCPDURL:     "/ContentDirectory.xml",
			ControlURL:  "/ctl/ContentDirectory",
			EventSubURL: "/evt/ContentDirectory",
		},
		embedServiceXML,
		map[string]upnp.ActionHandlerFunc{
			"Browse": h.Browse,
		})
	ctl.State.SetVariable("SystemUpdateID", "10")
	return ctl
}

type MyHandler struct {
}

func (ctl *MyHandler) Browse(ctx *upnp.ActionContext) error {

	ctx.Action.ArgOut.Set("TotalMatches", "4")
	ctx.Action.ArgOut.Set("NumberReturned", "4")
	ctx.Action.ArgOut.Set("UpdateID", ctx.State.GetVariable("SystemUpdateID"))
	ctx.Action.ArgOut.Set("Result", `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/"
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
	</DIDL-Lite>`)

	return nil
}
