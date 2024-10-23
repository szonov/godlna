package logger

import (
	"github.com/szonov/godlna/internal/soap"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"reflect"
)

func DebugPointer(message string, arg any) {
	debugArgs := make([]any, 0)

	if arg != nil {
		rv := reflect.ValueOf(arg)
		switch rv.Kind() {
		case reflect.Pointer:
			if rv.IsValid() && rv.Elem().Kind() == reflect.Struct {
				for i := 0; i < rv.Elem().NumField(); i++ {
					debugArgs = append(debugArgs,
						rv.Elem().Type().Field(i).Name,
						rv.Elem().Field(i).Interface(),
					)
				}
			}
		default:
			debugArgs = append(debugArgs, "arg", arg)
		}
	}
	slog.Debug(message, debugArgs...)
}

func DebugRequest(r *http.Request, headerBody ...bool) {
	var showHeader bool
	var showBody bool
	if len(headerBody) > 0 {
		showHeader = headerBody[0]
	}
	if len(headerBody) > 1 {
		showBody = headerBody[1]
	}

	attrs := []any{
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("remote", r.RemoteAddr),
		slog.String("request_id", r.Header.Get("X-Request-Id")),
	}

	if a := soap.DetectAction(r.Header.Get("SoapAction")); a != nil {
		attrs = append(attrs, slog.String("soap_action", a.Name))
	}

	slog.Debug("Request", attrs...)

	if showHeader {
		reqDump, _ := httputil.DumpRequest(r, showBody)
		slog.Debug("\n" + string(reqDump))
	}
}
