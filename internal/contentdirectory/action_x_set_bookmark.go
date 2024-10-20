package contentdirectory

import (
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/logger"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/internal/store"
	"github.com/szonov/godlna/internal/upnpav"
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

	logger.DebugPointer("X_SetBookmark [IN]", in)

	object := store.GetObject(in.ObjectID)
	if object == nil {
		soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
		return
	}
	if client.GetFeatures(r).UseSecondsInBookmark {
		in.PosSecond *= 1000
	}
	object.SetBookmark(in.PosSecond)
	soap.SendActionResponse(soapAction, nil, w)
}
