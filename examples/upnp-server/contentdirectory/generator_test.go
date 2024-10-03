package contentdirectory

import (
	"fmt"
	"github.com/szonov/go-upnp-lib/handler"
	"github.com/szonov/go-upnp-lib/scpd"
	"io"
	"os"
	"testing"
)

func loadScpdFromFile(file string) (*scpd.SCPD, error) {
	fp, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = fp.Close()
	}()
	xmlData, err := io.ReadAll(fp)
	if err != nil {
		return nil, err
	}
	serviceSCPD := new(scpd.SCPD)
	return serviceSCPD, serviceSCPD.Load(xmlData)
}

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
		CreateHandlerFile: "handler.go",
	}

	if err = serviceGen.GenerateService(); err != nil {
		t.Error(err)
	}

	fmt.Println("Success")
}
