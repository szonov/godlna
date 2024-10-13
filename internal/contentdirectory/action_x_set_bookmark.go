package contentdirectory

import (
	"github.com/szonov/godlna/internal/soap"
	"log/slog"
	"net/http"
)

type argInSetBookmark struct {
	CategoryType string
	RID          string
	ObjectID     string
	PosSecond    string
}

func actionSetBookmark(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	// input
	in := &argInSetBookmark{}
	if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
		soap.SendError(err, w)
		return
	}

	// todo: set bookmark
	slog.Debug("actionSetBookmark", slog.Any("in", in))

	soap.SendActionResponse(soapAction, nil, w)
}
