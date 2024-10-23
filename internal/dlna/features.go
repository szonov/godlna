package dlna

import (
	"net/http"
	"strings"
)

func UseSecondsInBookmark(r *http.Request) bool {
	agent := r.Header.Get("User-Agent")
	return agent == "DLNADOC/1.50" || strings.Contains(agent, "40C7000")
}
