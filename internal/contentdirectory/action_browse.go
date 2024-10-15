package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/client"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/internal/upnpav"
	"net/http"
	"strings"
)

type argInBrowse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int64
	RequestedCount int64
	SortCriteria   string
}

type argOutBrowse struct {
	Result         *soap.DIDLLite
	NumberReturned int
	TotalMatches   uint64
	UpdateID       uint64
}

func actionBrowse(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	in := &argInBrowse{}
	out := &argOutBrowse{}

	if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
		soap.SendError(err, w)
		return
	}

	profile := client.GetProfileByRequest(r)
	var objects []*backend.Object

	switch in.BrowseFlag {
	case "BrowseDirectChildren":
		objects, out.TotalMatches = backend.GetObjects(backend.ObjectFilter{
			ParentID: in.ObjectID,
			Limit:    in.RequestedCount,
			Offset:   in.StartingIndex,
		})
	case "BrowseMetadata":
		objects, out.TotalMatches = backend.GetObjects(backend.ObjectFilter{
			ObjectID: in.ObjectID,
			Limit:    1,
			Offset:   0,
		})
		if out.TotalMatches == 0 {
			soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
			return
		}
	default:
		err := fmt.Errorf("invalid BrowseFlag: %s", in.BrowseFlag)
		soap.SendUPnPError(soap.ArgumentValueInvalidErrorCode, err.Error(), w)
		return
	}

	out.NumberReturned = len(objects)
	out.UpdateID = serviceState.GetUint64("SystemUpdateID")
	out.Result = &soap.DIDLLite{
		Debug: strings.Contains(r.UserAgent(), "DIDLDebug"),
	}

	for _, object := range objects {
		item, err := transformObject(object, profile)
		if err != nil {
			soap.SendError(err, w)
			return
		}
		out.Result.Append(item)
	}

	soap.SendActionResponse(soapAction, out, w)
}
