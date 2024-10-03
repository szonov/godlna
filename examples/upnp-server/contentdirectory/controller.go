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
	ctl := &ServiceController{
		Service: &device.Service{
			ServiceType: ServiceType,
			ServiceId:   ServiceId,
			SCPDURL:     "/ContentDirectory.xml",
			ControlURL:  "/ctl/ContentDirectory",
			EventSubURL: "/evt/ContentDirectory",
		},
	}
	return ctl.createHandler()
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
		ctl.Handler.HandleSCPDURL(handler.NewHttpContext(w, r))
		return true
	}

	if r.URL.Path == ctl.Service.ControlURL {
		ctl.Handler.HandleControlURL(handler.NewHttpContext(w, r))
		return true
	}

	if r.URL.Path == ctl.Service.EventSubURL {
		ctl.Handler.HandleEventSubURL(handler.NewHttpContext(w, r))
		return true
	}

	return false
}

func (ctl *ServiceController) GetSearchCapabilities(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInGetSearchCapabilities)
	//out := action.ArgOut.(*ArgOutGetSearchCapabilities)
	return nil
}
func (ctl *ServiceController) GetSortCapabilities(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInGetSortCapabilities)
	//out := action.ArgOut.(*ArgOutGetSortCapabilities)
	return nil
}
func (ctl *ServiceController) GetSystemUpdateID(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInGetSystemUpdateID)
	//out := action.ArgOut.(*ArgOutGetSystemUpdateID)
	return nil
}
func (ctl *ServiceController) Browse(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInBrowse)
	//out := action.ArgOut.(*ArgOutBrowse)
	return nil
}
func (ctl *ServiceController) Search(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInSearch)
	//out := action.ArgOut.(*ArgOutSearch)
	return nil
}
func (ctl *ServiceController) CreateObject(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInCreateObject)
	//out := action.ArgOut.(*ArgOutCreateObject)
	return nil
}
func (ctl *ServiceController) DestroyObject(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInDestroyObject)
	//out := action.ArgOut.(*ArgOutDestroyObject)
	return nil
}
func (ctl *ServiceController) UpdateObject(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInUpdateObject)
	//out := action.ArgOut.(*ArgOutUpdateObject)
	return nil
}
func (ctl *ServiceController) ImportResource(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInImportResource)
	//out := action.ArgOut.(*ArgOutImportResource)
	return nil
}
func (ctl *ServiceController) ExportResource(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInExportResource)
	//out := action.ArgOut.(*ArgOutExportResource)
	return nil
}
func (ctl *ServiceController) StopTransferResource(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInStopTransferResource)
	//out := action.ArgOut.(*ArgOutStopTransferResource)
	return nil
}
func (ctl *ServiceController) GetTransferProgress(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInGetTransferProgress)
	//out := action.ArgOut.(*ArgOutGetTransferProgress)
	return nil
}
func (ctl *ServiceController) DeleteResource(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInDeleteResource)
	//out := action.ArgOut.(*ArgOutDeleteResource)
	return nil
}
func (ctl *ServiceController) CreateReference(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInCreateReference)
	//out := action.ArgOut.(*ArgOutCreateReference)
	return nil
}
