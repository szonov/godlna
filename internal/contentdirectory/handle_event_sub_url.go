package contentdirectory

import (
	"net/http"
)

func HandleEventSubURL(w http.ResponseWriter, r *http.Request) {
	eventManager.HandleEventSubURL(w, r, func() map[string]string {
		return map[string]string{
			"SystemUpdateID":     systemUpdateId,
			"ContainerUpdateIDs": "",
		}
	})
}
