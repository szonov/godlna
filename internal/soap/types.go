package soap

import (
	"encoding/xml"
	"log/slog"
	"reflect"
	"strings"
)

var (
	marshalerType = reflect.TypeFor[Marshaler]()
)

type (
	Marshaler interface {
		MarshalSoap() string
	}

	DIDLLite struct {
		Debug bool
		Items []interface{}
	}

	XMLLite string
)

func (v *XMLLite) String() string {
	return string(*v)
}

func (v *XMLLite) MarshalSoap() string {
	return xmlValueEncodeLight(string(*v))
}

func (v *DIDLLite) Append(item interface{}) {
	v.Items = append(v.Items, item)
}

func (v *DIDLLite) MarshalSoap() string {
	if v.Debug {
		return v.XMLBody()
	}
	return xmlValueEncodeLight(v.XMLBody())
}

func (v *DIDLLite) XMLBody() string {
	result, err := xml.Marshal(v.Items)
	if err != nil {
		slog.Error(err.Error())
	}
	return `<DIDL-Lite` +
		` xmlns:dc="http://purl.org/dc/elements/1.1/"` +
		` xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/"` +
		` xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"` +
		` xmlns:sec="http://www.sec.co.kr/"` +
		` xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/">` +
		string(result) +
		`</DIDL-Lite>`
}

func xmlValueEncodeLight(s string) string {
	res := strings.Replace(s, "<", "&lt;", -1)
	res = strings.Replace(res, ">", "&gt;", -1)
	return res
}
