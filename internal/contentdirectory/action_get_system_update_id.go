package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

func actionGetSystemUpdateID(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	updateID := serviceState.GetUint64("SystemUpdateID")

	soap.SendActionResponse(soapAction, fmt.Sprintf("<Id>%d</Id>", updateID), w)

}
