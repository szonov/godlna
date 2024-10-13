package contentdirectory

import (
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

type argOutGetSortCapabilities struct {
	SortCaps string
}

func actionGetSortCapabilities(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	out := &argOutGetSortCapabilities{
		SortCaps: "",
	}
	soap.SendActionResponse(soapAction, out, w)
}
