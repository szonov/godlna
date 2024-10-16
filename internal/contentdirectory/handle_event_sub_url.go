package contentdirectory

import (
	"github.com/szonov/godlna/internal/backend"
	"net/http"
)

func HandleEventSubURL(w http.ResponseWriter, r *http.Request) {

	if r.Method == "SUBSCRIBE" {
		res := eventManager.Subscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
			r.Header.Get("TIMEOUT"),
		)
		if res.Success {
			w.Header()["SID"] = []string{res.SID}
			w.Header()["TIMEOUT"] = []string{res.TimeoutHeaderString}
		}
		w.WriteHeader(res.StatusCode)

		if res.Success && res.IsNewSubscription {
			eventManager.SendInitialState(res.SID, map[string]string{
				"SystemUpdateID":     backend.GetSystemUpdateId().String(),
				"ContainerUpdateIDs": "",
			})
		}

	} else if r.Method == "UNSUBSCRIBE" {
		statusCode := eventManager.Unsubscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
		)
		w.WriteHeader(statusCode)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
