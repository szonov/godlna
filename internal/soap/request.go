package soap

import (
	"encoding/xml"
	"io"
)

type (
	envelopeReqBody struct {
		Request []byte `xml:",innerxml"`
	}

	envelopeReq struct {
		XMLName xml.Name        `xml:"Envelope"`
		Body    envelopeReqBody `xml:"Body"`
	}
)

func UnmarshalEnvelopeRequest(r io.Reader, args interface{}) (err error) {
	var env envelopeReq
	decoder := xml.NewDecoder(r)
	if err = decoder.Decode(&env); err == nil {
		err = xml.Unmarshal(env.Body.Request, args)
	}
	return
}
