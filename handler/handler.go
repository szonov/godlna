package handler

import (
	"encoding/xml"
	"fmt"
	"github.com/szonov/go-upnp-lib/scpd"
	"github.com/szonov/go-upnp-lib/soap"
	"net/http"
	"strconv"
)

const ResponseContentTypeXML = `text/xml; charset="utf-8"`

type ActionFunc func(ctx *Context) error
type ArgsFunc func() (in any, out any)

type Action struct {
	Name string
	Func ActionFunc
	Args ArgsFunc
}

type Context struct {
	Action  string
	Service string
	ArgIn   any
	ArgOut  any
	w       http.ResponseWriter
	r       *http.Request
	cancel  bool
}

func (c *Context) Writer() http.ResponseWriter {
	return c.w
}

func (c *Context) Request() *http.Request {
	return c.r
}

// Cancel gives possibility to send custom response from controllers action without sending ctx.ArgOut
func (c *Context) Cancel() {
	c.cancel = true
}

func (c *Context) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "u:" + c.Action + "Response"
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "xmlns:u"}, Value: c.Service},
	}
	return e.EncodeElement(c.ArgOut, start)
}

type Handler struct {
	Service string
	Actions []Action
	xmlBody []byte
}

func (h *Handler) Init() (err error) {
	var serviceSCPD *scpd.SCPD
	if serviceSCPD, err = MakeSCPD(h); err != nil {
		return
	}
	var xmlBody []byte
	if xmlBody, err = xml.Marshal(serviceSCPD); err != nil {
		return
	}
	h.xmlBody = append([]byte(xml.Header), xmlBody...)
	return
}

func (h *Handler) HandleSCPDURL(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method == http.MethodGet || method == http.MethodHead {
		h.sendXmlResponse(h.xmlBody, w)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h *Handler) HandleControlURL(w http.ResponseWriter, r *http.Request) {
	// scpd control URL works only with POST http method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// resolve current action name from http header
	soapAction := soap.DetectAction(r.Header.Get("SoapAction"))
	if soapAction == nil || soapAction.Service != h.Service {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// detect handler's action
	action, err := h.detectAction(soapAction.Name)
	if err != nil {
		h.sendError(soap.NewUPnPError(soap.InvalidActionErrorCode, err), w, http.StatusUnauthorized)
		return
	}

	// prepare context
	argIn, argOut := action.Args()
	ctx := &Context{
		Action:  soapAction.Name,
		Service: soapAction.Service,
		ArgIn:   argIn,
		ArgOut:  argOut,
		w:       w,
		r:       r,
	}

	// unmarshal request
	if err = soap.UnmarshalEnvelopeBody(r.Body, ctx.ArgIn); err != nil {
		h.sendError(soap.NewUPnPError(soap.ArgumentValueInvalidErrorCode, err), w, http.StatusBadRequest)
		return
	}

	// handle action
	if err = action.Func(ctx); err != nil {
		h.sendError(err, w, http.StatusInternalServerError)
		return
	}

	// send success response
	if !ctx.cancel {
		h.sendContext(ctx)
	}
}

func (h *Handler) HandleEventSubURL(w http.ResponseWriter, r *http.Request) {
	// todo: HandleEventSubURL
	h.sendError(fmt.Errorf("not implemented"), w, http.StatusNotImplemented)
}

func (h *Handler) detectAction(name string) (Action, error) {
	for _, action := range h.Actions {
		if action.Name == name {
			return action, nil
		}
	}
	return Action{}, fmt.Errorf("unknown action '%s'", name)
}

func (h *Handler) sendXmlResponse(xmlBody []byte, w http.ResponseWriter, statusCode ...int) {
	w.Header().Set("Content-Type", ResponseContentTypeXML)
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlBody)))
	w.Header().Set("EXT", "")
	if len(statusCode) > 0 {
		w.WriteHeader(statusCode[0])
	}
	_, _ = w.Write(xmlBody)
}

func (h *Handler) sendEnvelope(env *soap.Envelope, w http.ResponseWriter, statusCode ...int) {
	body, err := xml.Marshal(env)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body = append([]byte(xml.Header), body...)
	h.sendXmlResponse(body, w, statusCode...)
}

func (h *Handler) sendContext(ctx *Context) {
	env := soap.NewEnvelope(ctx)
	h.sendEnvelope(env, ctx.w)
}

func (h *Handler) sendError(err error, w http.ResponseWriter, statusCode ...int) {
	env := soap.NewErrEnvelope(soap.NewFailed(err))
	h.sendEnvelope(env, w, statusCode...)
}
