package contentdirectory

import (
	"github.com/szonov/godlna/pkg/soap"
	"net/http"
)

func actionGetSearchCapabilities(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	soap.SendActionResponse(soapAction, "<SearchCaps></SearchCaps>", w)

}
