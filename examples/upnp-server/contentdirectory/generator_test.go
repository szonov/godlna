package contentdirectory

import (
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
		Directory:         "cds",
		ControllerName:    "ServiceController",
		ControllerFile:    "controller.go",
		ArgumentsFile:     "arguments.go",
		CreateHandlerFile: "handlers.go",
	}

	if err = serviceGen.GenerateService(); err != nil {
		t.Error(err)
	}

	fmt.Println("Success")
}
