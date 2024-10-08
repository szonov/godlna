package service

import (
	"encoding/xml"
	"fmt"
	"io"
)

type (
	Action struct {
		Name        string
		ServiceType string
		ArgIn       ActionArgs
		ArgOut      ActionArgs
	}
	ActionArgs map[string]string
)

func (a ActionArgs) Set(name string, value string) {
	if _, ok := a[name]; ok {
		a[name] = value
		return
	}
}

func (a ActionArgs) Get(name string) string {
	return a[name]
}

func (a *Action) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if start.Name.Local != a.Name {
		return fmt.Errorf("unexpected envelope body, expect '%s', got '%s'",
			a.Name, start.Name.Local)
	}
	a.ServiceType = start.Name.Space
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		switch token := t.(type) {
		case xml.StartElement:
			var value string
			if err = d.DecodeElement(&value, &start); err != nil {
				return fmt.Errorf("xml encode problem '%s/%s'", start.Name.Local, token.Name.Local)
			}
			a.ArgIn[token.Name.Local] = value
		}
	}
	return nil
}

func (a *Action) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	start.Name.Local = "u:" + a.Name + "Response"
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "xmlns:u"}, Value: a.ServiceType},
	}
	if err = e.EncodeToken(start); err != nil {
		return
	}
	for k, v := range a.ArgOut {
		if err = e.EncodeElement(v, xml.StartElement{Name: xml.Name{Local: k}}); err != nil {
			return err
		}
	}
	err = e.EncodeToken(xml.EndElement{Name: start.Name})
	return
}
