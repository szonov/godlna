package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

func actionGetSystemUpdateID(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {

	soap.SendActionResponse(soapAction, fmt.Sprintf("<Id>%s</Id>", backend.GetSystemUpdateId().String()), w)

}
