package contentdirectory

import (
	"github.com/szonov/godlna/pkg/soap"
	"net/http"
)

func HandleSCPDURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		soap.SendXML(serviceDescriptionXML, w)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
