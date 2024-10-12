package soap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
)

const (
	InvalidActionErrorCode        uint = 401
	ActionFailedErrorCode         uint = 501
	ArgumentValueInvalidErrorCode uint = 600
)

type UPnPError struct {
	Code uint   `xml:"errorCode"`
	Desc string `xml:"errorDescription"`
}

func (e *UPnPError) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Desc)
}

func SendError(err error, w http.ResponseWriter, statusCode ...int) {
	code := ActionFailedErrorCode
	desc := err.Error()
	var upnpErr *UPnPError
	if errors.As(err, &upnpErr) {
		code = upnpErr.Code
		desc = upnpErr.Desc
	}
	body := xml.Header +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" xmlns:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body>` +
		`<s:Fault>` +
		`<faultcode>s:Client</faultcode>` +
		`<faultstring>UPnPError</faultstring>` +
		`<detail>` +
		`<UPnPError xmlns="urn:schemas-upnp-org:control-1-0">` +
		fmt.Sprintf(`<errorCode>%d</errorCode>`, code) +
		fmt.Sprintf(`<errorDescription>%s</errorDescription>`, xmlValueEncodeLight(desc)) +
		`</UPnPError>` +
		`</detail>` +
		`</s:Fault>` +
		`</s:Body>` +
		`</s:Envelope>`

	if len(statusCode) == 0 {
		statusCode = append(statusCode, http.StatusInternalServerError)
	}

	SendXML([]byte(body), w, statusCode...)
}

func SendUPnPError(code uint, desc string, w http.ResponseWriter, statusCode ...int) {
	SendError(&UPnPError{Code: code, Desc: desc}, w, statusCode...)
}
