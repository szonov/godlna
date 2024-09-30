package contentdirectory

import (
	_ "embed"
	"fmt"
	"github.com/szonov/go-upnp-lib/scpd"
)

//go:embed scpd.xml
var ServiceSCPDXML []byte
var ServiceSCPD *scpd.SCPD

func InitSCPD() {
	ServiceSCPD = new(scpd.SCPD)
	if err := ServiceSCPD.Load(ServiceSCPDXML); err != nil {
		panic(fmt.Errorf("invalid SCPD provided: %s", err))
	}
}
