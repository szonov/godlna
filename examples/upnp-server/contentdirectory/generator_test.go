package contentdirectory

import (
	"encoding/xml"
	"fmt"
	"github.com/szonov/go-upnp-lib/handler"
	"github.com/szonov/go-upnp-lib/scpd"
	"testing"
)

func TestGenerateService(t *testing.T) {
	var err error

	serviceSCPD := new(scpd.SCPD)
	if err = serviceSCPD.LoadFile("./generator_template.xml"); err != nil {
		t.Fatal(err)
	}

	serviceGen := &handler.ServiceGen{
		ServiceSCPD:       serviceSCPD,
		ServiceType:       "urn:schemas-upnp-org:service:ContentDirectory:1",
		ServiceId:         "urn:upnp-org:serviceId:ContentDirectory",
		Directory:         ".",
		ControllerName:    "ServiceController",
		ControllerFile:    "controller.go",
		ArgumentsFile:     "arguments.go",
		CreateHandlerFile: "handlers.go",
	}

	if err = serviceGen.GenerateService(); err != nil {
		t.Error(err)
	}
}

func TestMakeSCPD(t *testing.T) {

	ctl := NewServiceController()
	s, err := handler.MakeSCPD(ctl.Handler)
	if err != nil {
		t.Error(err)
	}
	var b []byte
	b, err = xml.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("SCPD: %s", b)
}
