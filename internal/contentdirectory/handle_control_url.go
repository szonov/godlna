package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

type (
	controlHandler func(*soap.Action, http.ResponseWriter, *http.Request)
)

var controlHandlers = map[string]controlHandler{
	"Browse":                actionBrowse,
	"GetSearchCapabilities": actionGetSearchCapabilities,
	"GetSortCapabilities":   actionGetSortCapabilities,
	"GetSystemUpdateID":     actionGetSystemUpdateID,
	"X_GetFeatureList":      actionGetFeatureList,
	"X_SetBookmark":         actionSetBookmark,
}

func HandleControlURL(w http.ResponseWriter, r *http.Request) {
	// Control URL works only with POST http method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// resolve current action name from http header
	soapAction := soap.DetectAction(r.Header.Get("SoapAction"))
	if soapAction == nil || soapAction.ServiceType != ServiceType {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f, ok := controlHandlers[soapAction.Name]
	if !ok {
		err := fmt.Errorf("unknown action '%s'", soapAction.Name)
		soap.SendUPnPError(soap.InvalidActionErrorCode, err.Error(), w, http.StatusUnauthorized)
		return
	}
	f(soapAction, w, r)
}
