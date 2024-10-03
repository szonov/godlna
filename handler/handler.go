package handler

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
	"strconv"
	"strings"
)

const ResponseContentTypeXML = `text/xml; charset="utf-8"`

type ActionCustomResponse struct {
}

func (a *ActionCustomResponse) Error() string {
	return fmt.Sprintf("controller action respond with own response, stop processing")
}

type ActionDefFunc func() *Action
type ActionMap map[string]ActionDefFunc
type ActionExecFunc func(action *Action) error

// Action definition of handler
type Action struct {
	// should be initialized before processing action
	f      ActionExecFunc
	ArgIn  any
	ArgOut any
	// added on action processing
	Name        string
	ServiceType string
	Ctx         HttpContext
}

// Stop gives possibility to return from controllers action custom error,
// and prevent handler from output anything (just stop operation)
func (a *Action) Stop() error {
	return &ActionCustomResponse{}
}

func (a *Action) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "u:" + a.Name + "Response"
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "xmlns:u"}, Value: a.ServiceType},
	}
	return e.EncodeElement(a.ArgOut, start)
}

func NewAction(f ActionExecFunc, in any, out any) *Action {
	return &Action{f: f, ArgIn: in, ArgOut: out}
}

type Handler struct {
	ServiceType string
	Actions     ActionMap
	xmlBody     []byte
}

func (h *Handler) Init() error {
	// TODO: validation, scpd generation
	h.xmlBody = []byte("TODO")
	return nil
}

func (h *Handler) HandleSCPDURL(ctx HttpContext) {
	method := ctx.GetMethod()
	if method == http.MethodGet || method == http.MethodHead {
		h.sendXmlResponse(ctx, h.xmlBody)
		return
	}
	ctx.Send(http.StatusMethodNotAllowed)
}

func (h *Handler) HandleControlURL(ctx HttpContext) {
	// scpd control URL works only with POST http method
	if ctx.GetMethod() != http.MethodPost {
		ctx.Send(http.StatusMethodNotAllowed)
		return
	}
	// resolve current action name from http header
	actionName := h.detectSoapAction(ctx)
	if actionName == "" {
		ctx.Send(http.StatusBadRequest)
		return
	}
	// detect handler
	actionDefFunc, ok := h.Actions[actionName]
	if !ok {
		err := soap.NewUPnPError(soap.InvalidActionErrorCode, fmt.Errorf("unknown action '%s'", actionName))
		h.sendError(ctx, err, http.StatusUnauthorized)
		return
	}
	// create action
	action := actionDefFunc()
	action.Name = actionName
	action.ServiceType = h.ServiceType
	action.Ctx = ctx
	// unmarshal request
	if err := soap.UnmarshalEnvelopeBody(ctx.GetBodyReader(), action.ArgIn); err != nil {
		h.sendError(ctx, soap.NewUPnPError(soap.ArgumentValueInvalidErrorCode, err), http.StatusBadRequest)
		return
	}
	// handle action
	if err := action.f(action); err != nil {
		if !errors.Is(err, &ActionCustomResponse{}) {
			h.sendError(ctx, err, http.StatusInternalServerError)
		}
		return
	}
	// send success response
	h.sendAction(ctx, action)
}

func (h *Handler) detectSoapAction(ctx HttpContext) string {
	header := strings.Trim(ctx.GetHeader("SoapAction"), " \"")
	parts := strings.Split(header, "#")
	if len(parts) == 2 && parts[0] == h.ServiceType && parts[1] != "" {
		return parts[1]
	}
	return ""
}

func (h *Handler) sendXmlResponse(ctx HttpContext, xmlBody []byte, statusCode ...int) {
	ctx.
		SetHeader("Content-Length", strconv.Itoa(len(xmlBody))).
		SetHeader("Content-Type", ResponseContentTypeXML).
		SetHeader("EXT", "").
		SetBody(xmlBody).
		Send(statusCode...)
}

func (h *Handler) sendEnvelope(ctx HttpContext, env *soap.Envelope, statusCode ...int) {
	body, err := xml.Marshal(env)
	if err != nil {
		ctx.Send(http.StatusInternalServerError)
		return
	}
	body = append([]byte(xml.Header), body...)
	h.sendXmlResponse(ctx, body, statusCode...)
}

func (h *Handler) sendAction(ctx HttpContext, action *Action, statusCode ...int) {
	env := soap.NewEnvelope(action)
	h.sendEnvelope(ctx, env, statusCode...)
}

func (h *Handler) sendError(ctx HttpContext, err error, statusCode ...int) {
	env := soap.NewErrEnvelope(soap.NewFailed(err))
	h.sendEnvelope(ctx, env, statusCode...)
}
