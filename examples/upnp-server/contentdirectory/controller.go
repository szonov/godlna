package contentdirectory

import (
	"github.com/szonov/go-upnp-lib"
	"net/http"
)

type ServiceController struct {
	service *upnp.Service
	s       *upnp.Server
}

func (c *ServiceController) OnServerStart(s *upnp.Server) error {
	c.service = &upnp.Service{
		ServiceType: ServiceType,
		ServiceId:   ServiceId,
		SCPDURL:     "/ContentDirectory.xml",
		ControlURL:  "/ctl/ContentDirectory",
		EventSubURL: "/evt/ContentDirectory",
	}
	c.s = s
	c.s.Device.ServiceList = append(s.Device.ServiceList, c.service)
	return nil
}

func (c *ServiceController) Handle(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == c.service.SCPDURL {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			w.Header().Set("Content-Type", upnp.ResponseContentTypeXML)
			_, _ = w.Write(ServiceSCPDXML)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return true
	}
	if r.URL.Path == c.service.ControlURL {
		if r.Method == http.MethodPost {
			// to do : work with soap
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return true
	}
	if r.URL.Path == c.service.EventSubURL {
		// events for later...
		w.WriteHeader(http.StatusNotImplemented)
	}
	return false
}
