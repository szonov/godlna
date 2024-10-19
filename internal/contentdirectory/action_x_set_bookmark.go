package contentdirectory

import (
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/logger"
	"github.com/szonov/godlna/internal/soap"
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

	if client.GetFeatures(r).UseSecondsInBookmark {
		in.PosSecond *= 1000
	}
	backend.SetBookmark(in.ObjectID, in.PosSecond)
	soap.SendActionResponse(soapAction, nil, w)
}
