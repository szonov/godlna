package upnpav

import (
	"encoding/xml"
	"fmt"
)

const (
	NoSuchObjectErrorCode = 701
)

// Resource description
type Resource struct {
	XMLName         xml.Name `xml:"res"`
	ProtocolInfo    string   `xml:"protocolInfo,attr"`
	URL             string   `xml:",chardata"`
	Size            int64    `xml:"size,attr,omitempty"`
	Bitrate         int      `xml:"bitrate,attr,omitempty"`
	Duration        string   `xml:"duration,attr,omitempty"`
	Resolution      string   `xml:"resolution,attr,omitempty"`
	AudioChannels   int      `xml:"nrAudioChannels,attr,omitempty"`
	SampleFrequency int      `xml:"sampleFrequency,attr,omitempty"`
}

// Container description
type Container struct {
	Object
	XMLName xml.Name `xml:"container"`
}

type Bookmark int64

func (b Bookmark) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("BM=%d", b)), nil
}

// Item description
type Item struct {
	Object
	XMLName  xml.Name `xml:"item"`
	Res      []Resource
	Bookmark Bookmark `xml:"sec:dcmInfo,omitempty"`
}

type AlbumArtURI struct {
	Profile string `xml:"dlna:profileID,attr,omitempty"`
	Value   string `xml:",chardata"`
}

// Object description
type Object struct {
	ID          string       `xml:"id,attr"`
	ParentID    string       `xml:"parentID,attr"`
	Restricted  int          `xml:"restricted,attr"`
	Title       string       `xml:"dc:title"`
	Class       string       `xml:"upnp:class"`
	Icon        string       `xml:"upnp:icon,omitempty"`
	Date        string       `xml:"dc:date,omitempty"`
	AlbumArtURI *AlbumArtURI `xml:"upnp:albumArtURI,omitempty"`
}
