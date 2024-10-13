package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/soap"
	"net/http"
)

type argOutGetFeatureList struct {
	FeatureList soap.XMLLite
}

func actionGetFeatureList(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	out := &argOutGetFeatureList{
		FeatureList: soap.XMLLite(
			`<Features xmlns="urn:schemas-upnp-org:av:avs" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` +
				` xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">` +
				`<Feature name="samsung.com_BASICVIEW" version="1">` +
				fmt.Sprintf(`<container id="%s" type="object.item.audioItem"/>`, backend.MusicID) +
				fmt.Sprintf(`<container id="%s" type="object.item.videoItem"/>`, backend.VideoID) +
				fmt.Sprintf(`<container id="%s" type="object.item.imageItem"/>`, backend.ImageID) +
				`</Feature>` +
				`</Features>`,
		),
	}
	soap.SendActionResponse(soapAction, out, w)
}
