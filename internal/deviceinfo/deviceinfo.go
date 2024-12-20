package deviceinfo

import (
	"embed"
	"encoding/xml"
	"fmt"
	"github.com/szonov/godlna/pkg/soap"
	"github.com/szonov/godlna/pkg/upnp/device"
	"io/fs"
	"net/http"
)

//go:embed icons
var embedIconsFS embed.FS

var (
	deviceDescXML  []byte
	indexHtml      []byte
	iconFileServer http.Handler
)

func Init(desc *device.Description) (err error) {
	if desc == nil {
		err = fmt.Errorf("device descirition can't be nil")
		return
	}
	if deviceDescXML, err = xml.Marshal(desc); err != nil {
		err = fmt.Errorf("marshal device desc error: '%s'", err.Error())
		return
	}
	deviceDescXML = append([]byte(xml.Header), deviceDescXML...)

	// embed file system with icons
	var sub fs.FS
	if sub, err = fs.Sub(embedIconsFS, "icons"); err != nil {
		err = fmt.Errorf("failed to load embedded icons fs: %w", err)
		return
	}
	iconFileServer = http.StripPrefix("/device/icons/", http.FileServer(http.FS(sub)))

	// index page - simple text
	indexHtml = []byte(fmt.Sprintf(`[%s] DLNA Server`, desc.Device.FriendlyName))

	return
}

func HandleDeviceDescriptionURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXML(deviceDescXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func HandlePresentationURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			_, _ = w.Write(indexHtml)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func HandleIcons(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		iconFileServer.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
