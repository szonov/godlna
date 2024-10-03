package scpd

import (
	"encoding/xml"
	"fmt"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
	"strconv"
	"strings"
)

const ResponseContentTypeXML = `text/xml; charset="utf-8"`

type HandlerActionDefFunc func() *HandlerAction
type HandlerActionMap map[string]HandlerActionDefFunc
type HandlerActionFunc func(action *HandlerAction) error

// HandlerAction definition of handler
type HandlerAction struct {
	// should be initialized before processing action
	f      HandlerActionFunc
	ArgIn  any
	ArgOut any
	// added on action processing
	Name        string
	ServiceType string
	Ctx         HttpContext
}

func (a *HandlerAction) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "u:" + a.Name + "Response"
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "xmlns:u"}, Value: a.ServiceType},
	}
	return e.EncodeElement(a.ArgOut, start)
}

func NewHandlerAction(f HandlerActionFunc, in any, out any) *HandlerAction {
	return &HandlerAction{f: f, ArgIn: in, ArgOut: out}
}

type Handler struct {
	ServiceType string
	Actions     HandlerActionMap
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
		h.sendError(ctx, err, http.StatusInternalServerError)
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
		SetHeader("Connection", "close").
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

func (h *Handler) sendAction(ctx HttpContext, action *HandlerAction, statusCode ...int) {
	env := soap.NewEnvelope(action)
	ctx.SetHeader("EXT", "")
	h.sendEnvelope(ctx, env, statusCode...)
}

func (h *Handler) sendError(ctx HttpContext, err error, statusCode ...int) {
	env := soap.NewErrEnvelope(soap.NewFailed(err))
	h.sendEnvelope(ctx, env, statusCode...)
}
