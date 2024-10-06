package presentation

import (
	"github.com/szonov/go-upnp-lib"
	"net/http"
)

type Controller struct {
}

func (ctl *Controller) RegisterRoutes(*upnp.DeviceDescription) ([]upnp.Route, error) {
	return []upnp.Route{
		{"/", ctl.handle},
	}, nil
}

func (ctl *Controller) handle(w http.ResponseWriter, r *http.Request) {
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
