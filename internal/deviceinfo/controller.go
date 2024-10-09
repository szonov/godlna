package deviceinfo

import (
	"embed"
	"encoding/xml"
	"fmt"
	"github.com/szonov/godlna/soap"
	"github.com/szonov/godlna/upnp/device"
	"io/fs"
	"net/http"
)

//go:embed icons
var embedIconsFS embed.FS

type Controller struct {
	descXML    []byte
	indexHtml  []byte
	fileServer http.Handler
}

func NewController(desc *device.Description) *Controller {

	sub, err := fs.Sub(embedIconsFS, "icons")
	if err != nil {
		panic(fmt.Errorf("failed to load embedded icons fs: %w", err))
	}
	fileServer := http.FileServer(http.FS(sub))

	b, _ := xml.Marshal(desc)
	return &Controller{
		descXML:    append([]byte(xml.Header), b...),
		indexHtml:  []byte(fmt.Sprintf(`[%s] DLNA Server`, desc.Device.FriendlyName)),
		fileServer: http.StripPrefix("/icons/", fileServer),
	}
}

func (ctl *Controller) HandleDescRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXML(ctl.descXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (ctl *Controller) HandlePresentationURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			_, _ = w.Write(ctl.indexHtml)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
func (ctl *Controller) HandleIcons(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		ctl.fileServer.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
