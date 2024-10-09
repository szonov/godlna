package contentdirectory1

import (
	_ "embed"
	"fmt"
	"github.com/szonov/godlna/soap"
	"github.com/szonov/godlna/upnp/events"
	"net/http"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

//go:embed scpd_impl.xml
var embedServiceXML []byte

type Controller struct {
	state *events.State
}

func NewController() *Controller {
	ctl := &Controller{
		state: events.NewState([]string{
			"TransferIDs",
			"SystemUpdateID",
			"ContainerUpdateIDs",
		}),
	}
	ctl.state.SetUint32("SystemUpdateID", 10)
	return ctl
}

func (ctl *Controller) HandleSCPDURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXML(embedServiceXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (ctl *Controller) HandleEventSubURL(w http.ResponseWriter, r *http.Request) {
	ctl.state.NetHttpEventSubURLHandler(w, r)
}

func (ctl *Controller) HandleControlURL(w http.ResponseWriter, r *http.Request) {
	// Control URL works only with POST http method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// resolve current action name from http header
	soapAction := soap.DetectAction(r.Header.Get("SoapAction"))
	if soapAction == nil || soapAction.ServiceType != ServiceType {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch soapAction.Name {
	case "Browse":
		ctl.actionBrowse(soapAction, w, r)
	default:
		err := fmt.Errorf("unknown action '%s'", soapAction.Name)
		soap.NewUPnPError(soap.InvalidActionErrorCode, err).SendResponse(w, http.StatusUnauthorized)
		return
	}
}

func (ctl *Controller) error(err error, w http.ResponseWriter) {
	soap.NewUPnPError(soap.InvalidActionErrorCode, err).SendResponse(w, http.StatusInternalServerError)
}
