package upnp

import (
	"encoding/xml"
	"fmt"
	"github.com/szonov/go-upnp-lib/events"
	"github.com/szonov/go-upnp-lib/scpd"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
	"strconv"
	"time"
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

	ctl := &ServiceController{
		service:        service,
		actions:        actions,
		stateVariables: new(events.PropertySet),
		subscribers:    new(events.Subscribers),
	}

	// build scpd, during it add stateVariables for arguments with SendEvents="yes"
	// do it on controller creation to make possibility to creators immediately set initial state variables
	xmlBody, err := ctl.buildScpdXml(ctl.actions)
	if err != nil {
		panic(err)
	}
	ctl.scpdXmlBody = append([]byte(xml.Header), xmlBody...)

	return ctl
}

func (ctl *ServiceController) Service() *Service {
	return ctl.service
}

func (ctl *ServiceController) RegisterRoutes(deviceDesc *DeviceDescription) ([]Route, error) {

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
	if r.Method == "SUBSCRIBE" {
		// subscribe
		sid := r.Header.Get("SID")
		nt := r.Header.Get("NT")
		callback := r.Header.Get("CALLBACK")
		timeout := events.ParseTimeoutHeader(r.Header.Get("TIMEOUT"))

		var subscriber *events.Subscriber
		isNewSubscriber := sid == ""

		if isNewSubscriber {
			// new subscriber
			if nt != "upnp:event" {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
			urls, err := events.ParseCallbackHeader(callback)
			if err != nil || len(urls) == 0 {
				w.WriteHeader(http.StatusPreconditionFailed)
			}
			sid = NewUDN(time.Now().String() + ":" + callback)
			subscriber = ctl.subscribers.Subscribe(sid, urls, timeout)
		} else {
			if nt != "" || callback != "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			subscriber = ctl.subscribers.Renew(sid, timeout)
		}
		if subscriber != nil {
			w.Header()["SID"] = []string{subscriber.SID}
			w.Header()["TIMEOUT"] = []string{"Second-" + strconv.Itoa(subscriber.Timeout)}

			if isNewSubscriber {
				// TODO: new subscription - should send initial state variables
			}
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if r.Method == "UNSUBSCRIBE" {
		// unsubscribe
		w.WriteHeader(ctl.subscribers.Unsubscribe(r.Header.Get("SID")))
	} else {
		// not expected
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

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
	serviceDesc := builder.SCPD()
	for _, st := range serviceDesc.Variables {
		if st.Events == "yes" {
			ctl.stateVariables.AddProperty(st.Name)
		}
	}
	body, err = xml.Marshal(serviceDesc)
	return
}
