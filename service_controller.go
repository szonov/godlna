package upnp

import (
	"fmt"
	"github.com/szonov/go-upnp-lib/scpd"
	"github.com/szonov/go-upnp-lib/service"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
)

type (
	ServiceController struct {
		service     *Service
		config      *service.Config
		State       *service.State
		scpdXmlBody []byte
		handlers    map[string]ActionHandlerFunc
	}

	ActionContext struct {
		Action *service.Action
		State  *service.State
		w      http.ResponseWriter
		r      *http.Request
		cancel bool
	}

	ActionHandlerFunc func(*ActionContext) error
)

func (ctx *ActionContext) Cancel() {
	ctx.cancel = true
}

func (ctx *ActionContext) Writer() http.ResponseWriter {
	return ctx.w
}

func (ctx *ActionContext) Request() *http.Request {
	return ctx.r
}

func NewServiceController(s *Service, scpdXmlBody []byte, handlers map[string]ActionHandlerFunc) *ServiceController {

	doc := &scpd.Document{}
	if err := doc.Load(scpdXmlBody); err != nil {
		panic(err)
	}

	cfg := service.NewConfig(s.ServiceType).FromDocument(doc)

	ctl := &ServiceController{
		service:     s,
		config:      cfg,
		State:       service.NewState(cfg.EventfulVariables()),
		scpdXmlBody: scpdXmlBody,
		handlers:    handlers,
	}
	return ctl
}

func (ctl *ServiceController) RegisterRoutes(deviceDesc *DeviceDescription) ([]Route, error) {

	deviceDesc.Device.AppendService(ctl.service)
	return []Route{
		{ctl.service.SCPDURL, ctl.handleSCPDURL},
		{ctl.service.ControlURL, ctl.handleControlURL},
		{ctl.service.EventSubURL, ctl.handleEventSubURL},
	}, nil
}

func (ctl *ServiceController) handleSCPDURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXmlResponse(ctl.scpdXmlBody, w)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (ctl *ServiceController) handleControlURL(w http.ResponseWriter, r *http.Request) {
	// Control URL works only with POST http method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// resolve current action name from http header
	soapAction := soap.DetectAction(r.Header.Get("SoapAction"))
	if soapAction == nil || soapAction.ServiceType != ctl.service.ServiceType {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	action := ctl.config.NewAction(soapAction.Name)
	if action == nil {
		err := fmt.Errorf("unknown action '%s'", soapAction.Name)
		soap.NewUPnPError(soap.InvalidActionErrorCode, err).SendResponse(w, http.StatusUnauthorized)
		return
	}

	// unmarshal request
	if err := soap.UnmarshalEnvelopeBody(r.Body, action); err != nil {
		soap.NewUPnPError(soap.ArgumentValueInvalidErrorCode, err).SendResponse(w, http.StatusBadRequest)
		return
	}

	ctx := &ActionContext{
		Action: action,
		State:  ctl.State,
		w:      w,
		r:      r,
	}

	// handle action
	if f, ok := ctl.handlers[action.Name]; ok {
		if err := f(ctx); err != nil {
			soap.NewFailed(err).SendResponse(w, http.StatusInternalServerError)
			return
		}
	}
	// todo: fallback

	// send success response
	if !ctx.cancel {
		soap.NewResponseEnvelope(action).SendResponse(w)
	}
}

func (ctl *ServiceController) handleEventSubURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == "SUBSCRIBE" {
		res := ctl.State.Subscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
			r.Header.Get("TIMEOUT"),
		)
		if res.Success {
			w.Header()["SID"] = []string{res.SID}
			w.Header()["TIMEOUT"] = []string{res.TimeoutHeaderString}
		}
		w.WriteHeader(res.StatusCode)
	} else if r.Method == "UNSUBSCRIBE" {
		statusCode := ctl.State.Unsubscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
		)
		w.WriteHeader(statusCode)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
