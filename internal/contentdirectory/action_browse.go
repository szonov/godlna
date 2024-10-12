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
	NumberReturned uint64
	TotalMatches   uint64
	UpdateID       uint64
}

func (ctl *Controller) actionBrowse(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	// input
	in := &argInBrowse{}
	if err := soap.UnmarshalEnvelopeRequest(r.Body, in); err != nil {
		soap.SendError(err, w)
		return
	}

	// output
	out := &argOutBrowse{}

	profile := client.GetProfileByRequest(r)
	var filter backend.ObjectFilter

	// I need only video, don't want to make extra click on TV to switch to Video folder
	// but for the future (and the second TV) the ability to change the behavior remains
	if profile.UseVideoAsRoot() && in.ObjectID == "0" {
		in.ObjectID = backend.VideoID
	}

	if in.BrowseFlag == "BrowseDirectChildren" {
		filter = backend.ObjectFilter{
			ParentID: in.ObjectID,
			Offset:   in.StartingIndex,
			Limit:    in.RequestedCount,
		}
	} else if in.BrowseFlag == "BrowseMetadata" {
		filter = backend.ObjectFilter{
			ObjectID: in.ObjectID,
			Offset:   0,
			Limit:    1,
		}
	} else {
		err := fmt.Errorf("invalid BrowseFlag: %s", in.BrowseFlag)
		soap.SendUPnPError(soap.ArgumentValueInvalidErrorCode, err.Error(), w)
		return
	}

	var objects []*backend.Object

	objects, out.TotalMatches = backend.GetObjects(filter)
	out.NumberReturned = uint64(len(objects))
	out.UpdateID = ctl.state.GetUint64("SystemUpdateID")

	if in.BrowseFlag == "BrowseMetadata" && out.TotalMatches == 0 {
		soap.SendUPnPError(upnpav.NoSuchObjectErrorCode, "no such object", w, http.StatusBadRequest)
		return
	}

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
