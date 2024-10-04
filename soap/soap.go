package soap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	EncodingStyle = "http://schemas.xmlsoap.org/soap/encoding/"
	EnvelopeNS    = "http://schemas.xmlsoap.org/soap/envelope/"

	InvalidActionErrorCode        uint = 401
	ActionFailedErrorCode         uint = 501
	ArgumentValueInvalidErrorCode uint = 600
)

type UPnPError struct {
	XMLName xml.Name `xml:"urn:schemas-upnp-org:control-1-0 UPnPError"`
	Code    uint     `xml:"errorCode"`
	Desc    string   `xml:"errorDescription"`
}

func (e *UPnPError) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Desc)
}

type FaultDetail struct {
	XMLName xml.Name `xml:"detail"`
	Data    interface{}
}

type Fault struct {
	XMLName     xml.Name    `xml:"s:Fault"`
	FaultCode   string      `xml:"faultcode"`
	FaultString string      `xml:"faultstring"`
	Detail      FaultDetail `xml:"detail"`
}

type EnvelopeBody struct {
	Request  []byte `xml:",innerxml"`
	Response any
}

type Envelope struct {
	XMLName xml.Name     `xml:"Envelope"`
	Body    EnvelopeBody `xml:"Body"`
}

func (v *Envelope) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	startEnvelope := xml.StartElement{
		Name: xml.Name{Local: "s:" + start.Name.Local},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:s"}, Value: EnvelopeNS},
			{Name: xml.Name{Local: "xmlns:encodingStyle"}, Value: EncodingStyle},
		},
	}

	if err := e.EncodeToken(startEnvelope); err != nil {
		return err
	}

	startBody := xml.StartElement{Name: xml.Name{Local: "s:Body"}}

	if err := e.EncodeElement(v.Body, startBody); err != nil {
		return err
	}

	return e.EncodeToken(xml.EndElement{Name: startEnvelope.Name})
}

func GetEnvelopeBody(r io.Reader) ([]byte, error) {
	var env Envelope
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&env); err != nil {
		return nil, err
	} else {
		return env.Body.Request, nil
	}
}

func UnmarshalEnvelopeBody(r io.Reader, args interface{}) (err error) {
	var body []byte
	if body, err = GetEnvelopeBody(r); err == nil {
		err = xml.Unmarshal(body, args)
	}
	return
}

func NewFailed(err error) *UPnPError {
	if err == nil {
		return nil
	}
	var e *UPnPError
	if errors.As(err, &e) {
		return e
	}
	return &UPnPError{Code: ActionFailedErrorCode, Desc: err.Error()}
}

func NewUPnPError(code uint, err error) *UPnPError {
	if err == nil {
		return nil
	}
	var e *UPnPError
	if errors.As(err, &e) {
		return e
	}
	return &UPnPError{Code: code, Desc: err.Error()}
}

func NewEnvelope(body any) *Envelope {
	return &Envelope{Body: EnvelopeBody{Response: body}}
}

func NewErrEnvelope(err error, faultString ...string) *Envelope {
	var s string
	if len(faultString) > 0 {
		s = faultString[0]
	}
	if s == "" {
		s = "UPnPError"
	}
	return NewEnvelope(&Fault{
		FaultCode:   "s:Client",
		FaultString: s,
		Detail: FaultDetail{
			Data: NewFailed(err),
		},
	})
}

func UnmarshalEnvelopeBody2[T any](r io.Reader, args T) (T, error) {
	var body []byte
	var err error
	if body, err = GetEnvelopeBody(r); err != nil {
		return args, err
	}
	err = xml.Unmarshal(body, args)
	return args, err
}

type Action struct {
	Name    string
	Service string
}

func DetectAction(soapActionHeader string) *Action {
	header := strings.Trim(soapActionHeader, " \"")
	parts := strings.Split(header, "#")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return &Action{
			Service: parts[0],
			Name:    parts[1],
		}
	}
	return nil
}
