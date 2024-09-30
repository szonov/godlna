package presentation

import (
	"github.com/szonov/go-upnp-lib"
	"net/http"
)

type DeviceInfoController struct {
	s *upnp.Server
}

func (c *DeviceInfoController) OnServerStart(s *upnp.Server) error {
	c.s = s
	c.s.Device.PresentationURL = "http://" + s.ListenAddress + "/"
	return nil
}

func (c *DeviceInfoController) Handle(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" {
		_, _ = w.Write([]byte("Index Page :: presentation URL"))
		return true
	}
	return false
}
