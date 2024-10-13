package contentdirectory

import (
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

type argOutGetSearchCapabilities struct {
	SearchCaps string
}

func actionGetSearchCapabilities(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	out := &argOutGetSearchCapabilities{
		SearchCaps: "",
	}
	soap.SendActionResponse(soapAction, out, w)
}
