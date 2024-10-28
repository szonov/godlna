package contentdirectory

import (
	"github.com/szonov/godlna/pkg/soap"
	"net/http"
)

func actionGetSortCapabilities(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	soap.SendActionResponse(soapAction, "<SortCaps></SortCaps>", w)

}
