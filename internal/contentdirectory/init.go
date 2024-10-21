package contentdirectory

import (
	"encoding/xml"
	"github.com/szonov/godlna/upnp/events"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

var (
	serviceDescriptionXML []byte
	eventManager          *events.Manager
	systemUpdateId        = "0"
)

func Init() (err error) {
	if serviceDescriptionXML, err = xml.Marshal(NewServiceDescription()); err != nil {
		return
	}
	serviceDescriptionXML = append([]byte(xml.Header), serviceDescriptionXML...)
	eventManager = events.NewManager()
	return
}
