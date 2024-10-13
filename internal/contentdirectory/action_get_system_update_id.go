package contentdirectory

import (
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

type argOutGetSystemUpdateID struct {
	Id uint64
}

func actionGetSystemUpdateID(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	out := &argOutGetSystemUpdateID{
		Id: serviceState.GetUint64("SystemUpdateID"),
	}
	soap.SendActionResponse(soapAction, out, w)
}
