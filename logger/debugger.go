package logger

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"reflect"
	"strings"
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
	}

	if actionName := soapActionName(r.Header.Get("SoapAction")); actionName != nil {
		attrs = append(attrs, slog.String("soap_action", *actionName))
	}

	slog.Debug("Request", attrs...)

	if showHeader {
		reqDump, _ := httputil.DumpRequest(r, showBody)
		slog.Debug("\n" + string(reqDump))
	}
}

func soapActionName(soapActionHeader string) *string {
	header := strings.Trim(soapActionHeader, " \"")
	parts := strings.Split(header, "#")
	if len(parts) == 2 && parts[1] != "" {
		return &parts[1]
	}
	return nil
}
