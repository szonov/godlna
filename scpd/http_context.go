package scpd

import (
	"io"
	"net/http"
)

// HttpContext define list of used functionality by scpd handler,
// keep in mind usage of other frameworks... fiber, gin, etc...
// scpd Handler do not use SetParam, GetParam at all, but it can help with
// realisation controllers, when defined methods no enough, for example
// for accessing to *http.Request
// before calling scpd handler:
//
//	ctx.SetParam("Request", r)
//
// then, in controllers action it can be received by:
//
//	r = action.Ctx.GetParam("Request").(*http.Request)
type HttpContext interface {
	SetParam(key string, value any) HttpContext
	GetParam(key string) any
	GetMethod() string
	GetHeader(name string) string
	SetHeader(name string, value string) HttpContext
	GetBodyReader() io.Reader
	SetBody(body []byte) HttpContext
	Send(statusCode ...int)
}

// HttpContextImpl implements HttpContext interface with build-in 'net/http'
type HttpContextImpl struct {
	w      http.ResponseWriter
	r      *http.Request
	body   []byte
	params map[string]any
}

func NewHttpContext(w http.ResponseWriter, r *http.Request) HttpContext {
	return &HttpContextImpl{w: w, r: r, params: make(map[string]any)}
}

func (ctx *HttpContextImpl) SetParam(key string, value any) HttpContext {
	ctx.params[key] = value
	return ctx
}

func (ctx *HttpContextImpl) GetParam(key string) any {
	if val, ok := ctx.params[key]; ok {
		return val
	}
	return nil
}

func (ctx *HttpContextImpl) GetMethod() string {
	return ctx.r.Method
}

func (ctx *HttpContextImpl) GetHeader(name string) string {
	return ctx.r.Header.Get(name)
}

func (ctx *HttpContextImpl) GetBodyReader() io.Reader {
	return ctx.r.Body
}

func (ctx *HttpContextImpl) SetHeader(name string, value string) HttpContext {
	ctx.w.Header().Set(name, value)
	return ctx
}

func (ctx *HttpContextImpl) SetBody(body []byte) HttpContext {
	ctx.body = body
	return ctx
}

func (ctx *HttpContextImpl) Send(statusCode ...int) {
	if len(statusCode) > 0 && statusCode[0] >= 200 && statusCode[0] < 600 {
		ctx.w.WriteHeader(statusCode[0])
	}
	_, _ = ctx.w.Write(ctx.body)
}
