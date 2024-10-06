package upnp

import (
	"encoding/xml"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
)

type DeviceController struct {
	descXML []byte
}

func (ctl *DeviceController) RegisterRoutes(deviceDesc *DeviceDescription) ([]Route, error) {

	// setup default location if not defined yet
	if deviceDesc.Location == "" {
		deviceDesc.Location = "/rootDesc.xml"
	}
	// make xml only once
	var b []byte
	var err error
	if b, err = xml.Marshal(deviceDesc); err != nil {
		return nil, err
	}
	ctl.descXML = append([]byte(xml.Header), b...)

	return []Route{
		{deviceDesc.Location, ctl.handle},
	}, nil
}

func (ctl *DeviceController) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXmlResponse(ctl.descXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
