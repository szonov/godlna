package scpd

import (
	"encoding/xml"
	"io"
	"os"
)

var Version = SpecVersion{
	Major: 1,
	Minor: 0,
}

type (
	Document struct {
		XMLName        xml.Name        `xml:"urn:schemas-upnp-org:service-1-0 scpd"`
		SpecVersion    SpecVersion     `xml:"specVersion"`
		Actions        []Action        `xml:"actionList>action"`
		StateVariables []StateVariable `xml:"serviceStateTable>stateVariable"`
	}

	SpecVersion struct {
		Major uint `xml:"major"`
		Minor uint `xml:"minor"`
	}

	StateVariable struct {
		Name              string             `xml:"name"`
		SendEvents        string             `xml:"sendEvents,attr,omitempty"`
		DataType          string             `xml:"dataType"`
		Default           string             `xml:"defaultValue,omitempty"`
		AllowedValues     *AllowedValueList  `xml:"allowedValueList,omitempty"`
		AllowedValueRange *AllowedValueRange `xml:"allowedValueRange,omitempty"`
	}

	AllowedValueList struct {
		Values []string `xml:"allowedValue"`
	}

	AllowedValueRange struct {
		Min  int `xml:"minimum"`
		Max  int `xml:"maximum"`
		Step int `xml:"step,omitempty"`
	}

	Action struct {
		Name string     `xml:"name"`
		Args []Argument `xml:"argumentList>argument"`
	}

	Argument struct {
		Name      string `xml:"name"`
		Direction string `xml:"direction"`
		Variable  string `xml:"relatedStateVariable"`
	}
)

func (doc *Document) Load(xmlData []byte) (err error) {
	err = xml.Unmarshal(xmlData, &doc)
	return
}

func (doc *Document) LoadFile(file string) (err error) {
	var fp *os.File
	var xmlData []byte

	if fp, err = os.Open(file); err != nil {
		return
	}
	defer func() {
		_ = fp.Close()
	}()

	if xmlData, err = io.ReadAll(fp); err != nil {
		return
	}
	err = doc.Load(xmlData)
	return
}
