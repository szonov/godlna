package events

import (
	"net/http"
	"time"
)

// HandleEventSubURL handles events sub url with 'net/http' build-in library
// f - function loader of current state variables, example
//
//	f := func() map[string]string {
//		return map[string]string{
//			"SystemUpdateID": getSystemUpdateId(),
//			"ContainerUpdateIDs": ContainerUpdateIDs(),
//		}
//	}
func (m *Manager) HandleEventSubURL(w http.ResponseWriter, r *http.Request, f func() map[string]string) {
	if r.Method == "SUBSCRIBE" {
		res := m.Subscribe(
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
			m.SendInitialState(res.SID, f(), 2*time.Second)
		}

	} else if r.Method == "UNSUBSCRIBE" {
		statusCode := m.Unsubscribe(
			r.Header.Get("SID"),
			r.Header.Get("NT"),
			r.Header.Get("CALLBACK"),
		)
		w.WriteHeader(statusCode)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
