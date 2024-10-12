package contentdirectory

import (
	_ "embed"
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/upnp/events"
	"log/slog"
	"net/http"
)

const (
	ServiceType = "urn:schemas-upnp-org:service:ContentDirectory:1"
	ServiceId   = "urn:upnp-org:serviceId:ContentDirectory"
)

//go:embed scpd.xml
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
	ctl.state.SetUint64("SystemUpdateID", backend.GetCurrentUpdateID())
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
	case "X_GetFeatureList":
		ctl.actionGetFeatureList(soapAction, w, r)
	default:
		err := fmt.Errorf("unknown action '%s'", soapAction.Name)
		soap.SendUPnPError(soap.InvalidActionErrorCode, err.Error(), w, http.StatusUnauthorized)
		return
	}
}

func (ctl *Controller) HandleThumbnailURL(w http.ResponseWriter, r *http.Request) {
	profile := client.GetProfileByRequest(r)
	image := r.PathValue("image")
	if image == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	slog.Debug("IMAGE", slog.Any("profile", profile), slog.String("image", image))
	soap.SendUPnPError(soap.InvalidActionErrorCode, image, w, http.StatusUnauthorized)
	return
}
