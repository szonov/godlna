package upnp

import (
	"encoding/xml"
	"fmt"
	"github.com/szonov/go-upnp-lib/events"
	"github.com/szonov/go-upnp-lib/scpd"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
)

// ActionHandlerFunc is the executor of action
type ActionHandlerFunc func(ctx *ActionContext) error

// ActionArgsFunc returns arguments in appropriate type for action handling
type ActionArgsFunc func() (in any, out any)

// ActionConfig is the configuration for action
type ActionConfig struct {
	// Name of action
	Name string
	// Func handles action
	Func ActionHandlerFunc
	// Args is a function which create input, output arguments during action handling
	Args ActionArgsFunc
}

// ActionContext passed to action handler function,
type ActionContext struct {
	Action soap.Action
	ArgIn  any
	ArgOut any
	// private
	ctl    *ServiceController
	cancel bool
	w      http.ResponseWriter
	r      *http.Request
}

func (ctx *ActionContext) Controller() *ServiceController {
	return ctx.ctl
}

func (ctx *ActionContext) Cancel() {
	ctx.cancel = true
	return
}
func (ctx *ActionContext) Writer() http.ResponseWriter {
	return ctx.w
}

func (ctx *ActionContext) Request() *http.Request {
	return ctx.r
}

func (ctx *ActionContext) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "u:" + ctx.Action.Name + "Response"
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "xmlns:u"}, Value: ctx.Action.ServiceType},
	}
	return e.EncodeElement(ctx.ArgOut, start)
}

type ServiceController struct {
	service        *Service
	actions        []ActionConfig
	stateVariables *events.PropertySet
	subscribers    *events.Subscribers
	scpdXmlBody    []byte
}

func NewServiceController(service *Service, actions []ActionConfig) *ServiceController {
	return &ServiceController{
		service:        service,
		actions:        actions,
		stateVariables: new(events.PropertySet),
		subscribers:    new(events.Subscribers),
	}
}

func (ctl *ServiceController) Service() *Service {
	return ctl.service
}

func (ctl *ServiceController) RegisterRoutes(deviceDesc *DeviceDescription) ([]Route, error) {
	// build scpd, during it add stateVariables for properties with SendEvents="yes"
	xmlBody, err := ctl.buildScpdXml(ctl.actions)
	if err != nil {
		return nil, err
	}
	ctl.scpdXmlBody = append([]byte(xml.Header), xmlBody...)

	deviceDesc.Device.AppendService(ctl.service)
	return []Route{
		{ctl.service.SCPDURL, ctl.handleSCPDURL},
		{ctl.service.ControlURL, ctl.handleControlURL},
		{ctl.service.EventSubURL, ctl.handleEventSubURL},
	}, nil
}

func (ctl *ServiceController) SetVariable(name string, value string, initialState ...bool) *ServiceController {
	ctl.stateVariables.Set(name, value, initialState...)
	return ctl
}

func (ctl *ServiceController) GetVariable(name string) string {
	return ctl.stateVariables.Get(name)
}

func (ctl *ServiceController) NotifySubscribers() {
	// events.
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

	// detect action configuration
	cfg := ctl.getActionConfig(soapAction.Name)
	if cfg == nil {
		err := fmt.Errorf("unknown action '%s'", soapAction.Name)
		soap.NewUPnPError(soap.InvalidActionErrorCode, err).SendResponse(w, http.StatusUnauthorized)
		return
	}

	// prepare context
	argIn, argOut := cfg.Args()
	ctx := &ActionContext{
		Action: *soapAction,
		ArgIn:  argIn,
		ArgOut: argOut,
		ctl:    ctl,
		w:      w,
		r:      r,
	}

	// unmarshal request
	if err := soap.UnmarshalEnvelopeBody(r.Body, ctx.ArgIn); err != nil {
		soap.NewUPnPError(soap.ArgumentValueInvalidErrorCode, err).SendResponse(w, http.StatusBadRequest)
		return
	}

	// handle action
	if err := cfg.Func(ctx); err != nil {
		soap.NewFailed(err).SendResponse(w, http.StatusInternalServerError)
		return
	}

	// send success response
	if !ctx.cancel {
		soap.NewResponseEnvelope(ctx).SendResponse(w)
	}
}

func (ctl *ServiceController) handleEventSubURL(w http.ResponseWriter, r *http.Request) {
	// todo
}

func (ctl *ServiceController) getActionConfig(actionName string) *ActionConfig {
	for _, action := range ctl.actions {
		if action.Name == actionName {
			return &action
		}
	}
	return nil
}

func (ctl *ServiceController) buildScpdXml(actions []ActionConfig) (body []byte, err error) {
	builder := scpd.NewBuilder(1, 0)
	for _, action := range actions {
		in, out := action.Args()
		if err = builder.Add(action.Name, in, out); err != nil {
			return
		}
	}
	body, err = xml.Marshal(builder.SCPD())
	return
}
