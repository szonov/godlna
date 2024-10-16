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
	profile := client.GetProfileByRequest(r)

	logger.DebugPointer("[input] X_SetBookmark", in)

	backend.SetBookmark(in.ObjectID, profile.BookmarkStoreValue(in.PosSecond))

	// notify subscribers
	updateId := backend.GetSystemUpdateId().String()
	eventManager.NotifyAll(map[string]string{
		"SystemUpdateID":     updateId,
		"ContainerUpdateIDs": backend.GetParentID(in.ObjectID) + "," + updateId,
	})

	soap.SendActionResponse(soapAction, nil, w)
}
