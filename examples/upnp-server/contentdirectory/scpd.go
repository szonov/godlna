package contentdirectory

import (
	_ "embed"
	"fmt"
	"github.com/szonov/go-upnp-lib/scpd"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

//go:embed scpd.xml
var ServiceSCPDXML []byte
var ServiceSCPD *scpd.SCPD

func InitSCPD() {
	if ServiceSCPD == nil {
		// initialize if and only if not initialized yet
		ServiceSCPD = new(scpd.SCPD)
		if err := ServiceSCPD.Load(ServiceSCPDXML); err != nil {
			panic(fmt.Errorf("invalid SCPD provided: %s", err))
		}
	}
}
