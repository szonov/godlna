package contentdirectory

import (
	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/device"
	"github.com/szonov/go-upnp-lib/scpd"
	"net/http"
)

type Controller struct {
	Handler *scpd.Handler
	Service *device.Service
}

func NewController() *Controller {
	ctl := &Controller{
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
func (ctl *Controller) OnServerStart(server *upnp.Server) error {
	if err := ctl.Handler.Init(); err != nil {
		return err
	}
	server.Device.ServiceList = append(server.Device.ServiceList, ctl.Service)
	return nil
}

// Handle implements upnp.Controller interface
func (ctl *Controller) Handle(w http.ResponseWriter, r *http.Request) bool {

	if r.URL.Path == ctl.Service.SCPDURL {
		ctl.Handler.HandleSCPDURL(scpd.NewHttpContext(w, r))
		return true
	}

	if r.URL.Path == ctl.Service.ControlURL {
		ctl.Handler.HandleControlURL(scpd.NewHttpContext(w, r))
		return true
	}

	// TODO: Eventing
	return false
}

func (ctl *Controller) Browse(action *scpd.HandlerAction) error {
	//in := action.ArgIn.(*ArgInBrowse)
	out := action.ArgOut.(*ArgOutBrowse)

	out.Result = "Result"
	out.NumberReturned = 10
	out.TotalMatches = 100
	out.UpdateID = 3

	//return fmt.Errorf("Test error")
	//return soap.NewUPnPError(300, fmt.Errorf("Test UPnP Error"))
	return nil
}
