package soap

import (
	"strings"
)

type (
	Action struct {
		ServiceType string
		Name        string
	}
)

func DetectAction(soapActionHeader string) *Action {
	header := strings.Trim(soapActionHeader, " \"")
	parts := strings.Split(header, "#")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return &Action{
			ServiceType: parts[0],
			Name:        parts[1],
		}
	}
	return nil
}
