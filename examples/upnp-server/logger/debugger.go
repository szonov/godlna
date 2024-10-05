package logger

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func DebugRequest(r *http.Request) {
	body := r.Method + " " + r.URL.String() + " " + r.Proto + "\r\n"
	for k, v := range r.Header {
		for _, vv := range v {
			body += fmt.Sprintf("%s: %s\r\n", k, vv)
		}
	}
	buf, err := io.ReadAll(r.Body) // handle the error
	if err == nil && len(buf) > 0 {
		body += "\r\n" + string(buf)
		rdr1 := io.NopCloser(bytes.NewBuffer(buf))
		r.Body = rdr1
	}
	slog.Debug("\n" + body)
}
