package contentdirectory

import (
	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/device"
	"github.com/szonov/go-upnp-lib/handler"
	"net/http"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

type ServiceController struct {
	Handler *handler.Handler
	Service *device.Service
}

func NewServiceController() *ServiceController {
	ctl := new(ServiceController)
	ctl.Service = &device.Service{
		ServiceType: ServiceType,
		ServiceId:   ServiceId,
		SCPDURL:     "/ContentDirectory.xml",
		ControlURL:  "/ctl/ContentDirectory",
		EventSubURL: "/evt/ContentDirectory",
	}
	ctl.Handler = &handler.Handler{
		Service: ctl.Service.ServiceType,
		Actions: ctl.createActions(),
	}
	return ctl
}

// OnServerStart implements upnp.Controller interface
func (ctl *ServiceController) OnServerStart(server *upnp.Server) error {
	if err := ctl.Handler.Init(); err != nil {
		return err
	}
	server.Device.ServiceList = append(server.Device.ServiceList, ctl.Service)
	return nil
}

// Handle implements upnp.Controller interface
func (ctl *ServiceController) Handle(w http.ResponseWriter, r *http.Request) bool {

	if r.URL.Path == ctl.Service.SCPDURL {
		ctl.Handler.HandleSCPDURL(w, r)
		return true
	}

	if r.URL.Path == ctl.Service.ControlURL {
		ctl.Handler.HandleControlURL(w, r)
		return true
	}

	if r.URL.Path == ctl.Service.EventSubURL {
		ctl.Handler.HandleEventSubURL(w, r)
		return true
	}

	return false
}

func (ctl *ServiceController) GetSearchCapabilities(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInGetSearchCapabilities)
	//out := ctx.ArgOut.(*ArgOutGetSearchCapabilities)
	return nil
}
func (ctl *ServiceController) GetSortCapabilities(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInGetSortCapabilities)
	//out := ctx.ArgOut.(*ArgOutGetSortCapabilities)
	return nil
}
func (ctl *ServiceController) GetSystemUpdateID(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInGetSystemUpdateID)
	//out := ctx.ArgOut.(*ArgOutGetSystemUpdateID)
	return nil
}
func (ctl *ServiceController) Browse(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInBrowse)
	//out := ctx.ArgOut.(*ArgOutBrowse)
	return nil
}
func (ctl *ServiceController) Search(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInSearch)
	//out := ctx.ArgOut.(*ArgOutSearch)
	return nil
}
func (ctl *ServiceController) CreateObject(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInCreateObject)
	//out := ctx.ArgOut.(*ArgOutCreateObject)
	return nil
}
func (ctl *ServiceController) DestroyObject(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInDestroyObject)
	//out := ctx.ArgOut.(*ArgOutDestroyObject)
	return nil
}
func (ctl *ServiceController) UpdateObject(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInUpdateObject)
	//out := ctx.ArgOut.(*ArgOutUpdateObject)
	return nil
}
func (ctl *ServiceController) ImportResource(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInImportResource)
	//out := ctx.ArgOut.(*ArgOutImportResource)
	return nil
}
func (ctl *ServiceController) ExportResource(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInExportResource)
	//out := ctx.ArgOut.(*ArgOutExportResource)
	return nil
}
func (ctl *ServiceController) StopTransferResource(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInStopTransferResource)
	//out := ctx.ArgOut.(*ArgOutStopTransferResource)
	return nil
}
func (ctl *ServiceController) GetTransferProgress(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInGetTransferProgress)
	//out := ctx.ArgOut.(*ArgOutGetTransferProgress)
	return nil
}
func (ctl *ServiceController) DeleteResource(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInDeleteResource)
	//out := ctx.ArgOut.(*ArgOutDeleteResource)
	return nil
}
func (ctl *ServiceController) CreateReference(ctx *handler.Context) error {
	//in := ctx.ArgIn.(*ArgInCreateReference)
	//out := ctx.ArgOut.(*ArgOutCreateReference)
	return nil
}
