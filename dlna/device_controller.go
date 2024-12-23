package dlna

import (
	"embed"
	"encoding/xml"
	"fmt"
	"github.com/szonov/godlna/soap"
	"io/fs"
	"net/http"
)

//go:embed icons
var embedIconsFS embed.FS

type DeviceController struct {
	deviceDescXML  []byte
	indexHtml      []byte
	iconFileServer http.Handler
}

func NewDeviceController(srv *Server) (*DeviceController, error) {
	var err error
	ctl := &DeviceController{}
	desc := srv.DeviceDescription
	if desc == nil {
		return ctl, fmt.Errorf("device descirition can't be nil")
	}

	if ctl.deviceDescXML, err = xml.Marshal(desc); err != nil {
		return ctl, fmt.Errorf("marshal device desc error: '%s'", err.Error())
	}
	ctl.deviceDescXML = append([]byte(xml.Header), ctl.deviceDescXML...)

	// embed file system with icons
	var sub fs.FS
	if sub, err = fs.Sub(embedIconsFS, "icons"); err != nil {
		return ctl, fmt.Errorf("failed to load embedded icons fs: %w", err)
	}
	ctl.iconFileServer = http.StripPrefix("/device/icons/", http.FileServer(http.FS(sub)))

	// index page - simple text
	ctl.indexHtml = []byte(fmt.Sprintf(`[%s] Video DLNA Server`, desc.Device.FriendlyName))

	return ctl, nil
}

func (ctl *DeviceController) HandleIndexURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			_, _ = w.Write(ctl.indexHtml)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (ctl *DeviceController) HandleDescriptionURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXML(ctl.deviceDescXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (ctl *DeviceController) HandleIcons(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		ctl.iconFileServer.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
