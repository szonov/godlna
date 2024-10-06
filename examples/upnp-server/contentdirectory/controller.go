package contentdirectory

import (
	"github.com/szonov/go-upnp-lib"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

func NewServiceController() *upnp.ServiceController {
	return upnp.NewServiceController(&upnp.Service{
		ServiceType: ServiceType,
		ServiceId:   ServiceId,
		SCPDURL:     "/ContentDirectory.xml",
		ControlURL:  "/ctl/ContentDirectory",
		EventSubURL: "/evt/ContentDirectory",
	}, new(ServiceController).actions())
}

type ServiceController struct {
}

func (ctl *ServiceController) GetSearchCapabilities(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInGetSearchCapabilities)
	//out := ctx.ArgOut.(*ArgOutGetSearchCapabilities)
	return nil
}
func (ctl *ServiceController) GetSortCapabilities(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInGetSortCapabilities)
	//out := ctx.ArgOut.(*ArgOutGetSortCapabilities)
	return nil
}
func (ctl *ServiceController) GetSystemUpdateID(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInGetSystemUpdateID)
	//out := ctx.ArgOut.(*ArgOutGetSystemUpdateID)
	return nil
}
func (ctl *ServiceController) Browse(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInBrowse)
	out := ctx.ArgOut.(*ArgOutBrowse)

	out.TotalMatches = 4
	out.NumberReturned = 4
	out.UpdateID = 1
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

	return nil
}
func (ctl *ServiceController) Search(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInSearch)
	//out := ctx.ArgOut.(*ArgOutSearch)
	return nil
}
func (ctl *ServiceController) CreateObject(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInCreateObject)
	//out := ctx.ArgOut.(*ArgOutCreateObject)
	return nil
}
func (ctl *ServiceController) DestroyObject(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInDestroyObject)
	//out := ctx.ArgOut.(*ArgOutDestroyObject)
	return nil
}
func (ctl *ServiceController) UpdateObject(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInUpdateObject)
	//out := ctx.ArgOut.(*ArgOutUpdateObject)
	return nil
}
func (ctl *ServiceController) ImportResource(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInImportResource)
	//out := ctx.ArgOut.(*ArgOutImportResource)
	return nil
}
func (ctl *ServiceController) ExportResource(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInExportResource)
	//out := ctx.ArgOut.(*ArgOutExportResource)
	return nil
}
func (ctl *ServiceController) StopTransferResource(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInStopTransferResource)
	//out := ctx.ArgOut.(*ArgOutStopTransferResource)
	return nil
}
func (ctl *ServiceController) GetTransferProgress(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInGetTransferProgress)
	//out := ctx.ArgOut.(*ArgOutGetTransferProgress)
	return nil
}
func (ctl *ServiceController) DeleteResource(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInDeleteResource)
	//out := ctx.ArgOut.(*ArgOutDeleteResource)
	return nil
}
func (ctl *ServiceController) CreateReference(ctx *upnp.ActionContext) error {
	//in := ctx.ArgIn.(*ArgInCreateReference)
	//out := ctx.ArgOut.(*ArgOutCreateReference)
	return nil
}
