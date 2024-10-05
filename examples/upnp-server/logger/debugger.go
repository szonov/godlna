package logger

import (
	"bytes"
	"fmt"
	"github.com/szonov/go-upnp-lib"
	"io"
	"log/slog"
	"net/http"
)

type DebugController struct {
	s *upnp.Server
}

func NewDebugController() *DebugController {
	return &DebugController{}
}

func (c *DebugController) OnServerStart(s *upnp.Server) error {
	return nil
}

func (c *DebugController) Handle(w http.ResponseWriter, r *http.Request) bool {

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

	return false
}
