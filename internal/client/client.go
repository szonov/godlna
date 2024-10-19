package client

import (
	"net/http"
	"strings"
)

type Features struct {
	UseSquareThumbnails  bool
	UseSecondsInBookmark bool
}

func GetFeatures(r *http.Request) *Features {
	agent := r.Header.Get("User-Agent")

	// here possible to check Remote IP
	// but in my environment there is only one special device that is uniquely identified by the user agent

	if agent == "DLNADOC/1.50" || strings.Contains(agent, "40C7000") {
		// User-Agent: SEC_HHP_TV-40C7000/1.0 (when get device description)
		// User-Agent: DLNADOC/1.50 (when POST content directory control url)
		return &Features{
			UseSquareThumbnails:  false,
			UseSecondsInBookmark: true,
		}
	}
	// all others... for example second TV:
	// User-Agent: DLNADOC/1.50 SEC_HHP_[TV] Samsung 5 Series (55)/1.0 UPnP/1.0
	return &Features{
		UseSquareThumbnails:  true,
		UseSecondsInBookmark: false,
	}
}
