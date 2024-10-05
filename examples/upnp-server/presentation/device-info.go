package presentation

import (
	"github.com/szonov/go-upnp-lib"
	"net/http"
)

type Controller struct {
	s *upnp.Server
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) OnServerStart(s *upnp.Server) error {
	s.DeviceDescription.Device.PresentationURL = "http://" + s.ListenAddress + "/"
	s.Handle("/", c.handleIndexPage)
	return nil
}

func (c *Controller) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method == "GET" || r.Method == "HEAD" {
			_, _ = w.Write([]byte("Index Page :: presentation URL"))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
