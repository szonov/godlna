package contentdirectory

import (
	"github.com/szonov/godlna/pkg/soap"
	"net/http"
)

type argOutGetFeatureList struct {
	FeatureList soap.XMLLite
}

func actionGetFeatureList(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	out := &argOutGetFeatureList{
		FeatureList: `<Features xmlns="urn:schemas-upnp-org:av:avs"` +
			` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` +
			` xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">` +
			`<Feature name="samsung.com_BASICVIEW" version="1">` +
			`<container id="0" type="object.item.videoItem"/>` +
			`</Feature>` +
			`</Features>`,
	}
	soap.SendActionResponse(soapAction, out, w)
}
