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

func (ctl *ServiceController) Browse(action *handler.Action) error {
	//in := action.ArgIn.(*ArgInBrowse)
	out := action.ArgOut.(*ArgOutBrowse)

	out.Result = "Result"
	out.NumberReturned = 10
	out.TotalMatches = 100
	out.UpdateID = 3

	return nil
}
