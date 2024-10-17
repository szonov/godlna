package client

import (
	"net/http"
	"strings"
)

const (
	DefaultProfile = "gen"
	Samsung4       = "sm4"
	Samsung5       = "sm5"
)

type Profile struct {
	Name string
}

func (p *Profile) DeviceDescriptionXML(deviceDescTemplate string) string {
	return strings.Replace(deviceDescTemplate, "{profile}", p.Name, -1)
}

func (p *Profile) UseSquareThumbnails() bool {
	return p.Name != Samsung4
}

func (p *Profile) UseBookmarkMilliseconds() bool {
	return p.Name == Samsung5
}

func GetProfileByRequest(r *http.Request) *Profile {
	p := &Profile{
		Name: DefaultProfile,
	}
	routeValue := r.PathValue("profile")
	if routeValue != "" {
		p.Name = routeValue
	} else {
		ua := r.Header.Get("User-Agent")
		if strings.Contains(ua, "40C7000") {
			//User-Agent: SEC_HHP_TV-40C7000/1.0
			p.Name = Samsung4
		} else if strings.Contains(ua, "Samsung 5 Series") {
			// User-Agent: DLNADOC/1.50 SEC_HHP_[TV] Samsung 5 Series (55)/1.0 UPnP/1.0
			p.Name = Samsung5
		}
	}
	return p
}
