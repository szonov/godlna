package contentdirectory

import (
	"github.com/szonov/godlna/internal/db"
	"github.com/szonov/godlna/internal/dlna"
	"github.com/szonov/godlna/internal/upnpav"
	"github.com/szonov/godlna/pkg/soap"
	"net/http"
)

type argInSetBookmark struct {
	CategoryType string
	RID          string
	ObjectID     string
	PosSecond    uint64
}

func actionSetBookmark(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	// input
	in := &argInSetBookmark{}
	if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
		soap.SendError(err, w)
		return
	}

	object := db.GetObject(in.ObjectID)
	if object == nil {
		soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
		return
	}
	if dlna.UseSecondsInBookmark(r) {
		in.PosSecond *= 1000
	}
	object.SetBookmark(in.PosSecond)
	soap.SendActionResponse(soapAction, nil, w)
}
