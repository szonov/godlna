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
)

func Init() (err error) {

	doc := NewServiceDescription()
	serviceDescriptionXML, err = xml.Marshal(doc)
	if err != nil {
		return
	}
	serviceDescriptionXML = append([]byte(xml.Header), serviceDescriptionXML...)

	eventfulVariables := make([]string, 0)
	for _, st := range doc.StateVariables {
		if st.SendEvents == "yes" {
			eventfulVariables = append(eventfulVariables, st.Name)
		}
	}
	eventManager = events.NewManager()
	doc = nil
	return
}
