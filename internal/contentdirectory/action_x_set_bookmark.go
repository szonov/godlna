package contentdirectory

import (
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
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
	backend.SetBookmark(in.ObjectID, client.GetProfileByRequest(r).BookmarkStoreValue(in.PosSecond))
	soap.SendActionResponse(soapAction, nil, w)
}
