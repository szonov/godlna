package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

const (
	ResponseContentTypeXML = `text/xml; charset="utf-8"`
)

func SendXML(xmlBody []byte, w http.ResponseWriter, statusCode ...int) {
	w.Header().Set("Content-Type", ResponseContentTypeXML)
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlBody)))
	if len(statusCode) > 0 {
		w.WriteHeader(statusCode[0])
	}

	_, err := w.Write(xmlBody)

	if err != nil {
		slog.Debug("XML write response failed ", slog.String("error", err.Error()))
	}
}

func encodeResValue(rv reflect.Value, b *bytes.Buffer, enc *xml.Encoder) string {
	// Check for marshaler.
	if rv.CanInterface() && rv.Type().Implements(marshalerType) {
		return rv.Interface().(Marshaler).MarshalSoap()
	}
	if rv.CanAddr() {
		pv := rv.Addr()
		if pv.CanInterface() && pv.Type().Implements(marshalerType) {
			return pv.Interface().(Marshaler).MarshalSoap()
		}
	}

	// use standard xml encoder => will get <x>[needed value]</x>
	if err := enc.EncodeElement(rv.Interface(), xml.StartElement{Name: xml.Name{Local: "x"}}); err != nil {
		return ""
	}
	bt := b.Bytes()
	b.Reset()
	return string(bt[3 : len(bt)-4])
}

func encodeResStruct(rv reflect.Value) string {
	response := ""
	var b bytes.Buffer
	enc := xml.NewEncoder(&b)

	for i := 0; i < rv.NumField(); i++ {
		tagName := rv.Type().Field(i).Name
		tag := rv.Type().Field(i).Tag.Get("xml")
		if tag != "" {
			tagName = strings.Split(tag, ",")[0]
		}
		response += fmt.Sprintf(`<%[1]s>%[2]s</%[1]s>`, tagName, encodeResValue(rv.Field(i), &b, enc))
	}
	_ = enc.Close()
	return response
}

func buildXML(res any) string {
	if res == nil {
		return ""
	}
	rv := reflect.ValueOf(res)
	switch rv.Kind() {
	case reflect.String:
		// ready XML as string
		return res.(string)
	case reflect.Slice:
		// ready XML as []byte type
		if slice, ok := rv.Interface().([]byte); ok {
			return string(slice)
		}
		slog.Error(fmt.Sprintf("unsupported response slice: '%s'", rv.Type()))
		return ""
	case reflect.Struct:
		// make XML from structure
		return encodeResStruct(rv)
	case reflect.Pointer:
		if rv.IsValid() && rv.Elem().Kind() == reflect.Struct {
			return encodeResStruct(rv.Elem())
		}
		slog.Error(fmt.Sprintf("invalid pointer: '%s'", rv.Type()))
		return ""
	default:
		slog.Error(fmt.Sprintf("unsupported response kind: '%s'", rv.Kind()))
		return ""
	}
}

func SendActionResponse(a *Action, response any, w http.ResponseWriter, statusCode ...int) {

	body := xml.Header +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" xmlns:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body>` +
		fmt.Sprintf(`<u:%sResponse xmlns:u="%s">`, a.Name, a.ServiceType) +
		buildXML(response) +
		fmt.Sprintf(`</u:%sResponse>`, a.Name) +
		`</s:Body>` +
		`</s:Envelope>`

	SendXML([]byte(body), w, statusCode...)
}
