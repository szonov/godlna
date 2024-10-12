package contentdirectory

import (
	"fmt"
	"github.com/szonov/godlna/internal/soap"
	"github.com/szonov/godlna/internal/storage"
	"net/http"
)

type argOutGetFeatureList struct {
	FeatureList soap.XMLLite
}

func (ctl *Controller) actionGetFeatureList(soapAction *soap.Action, w http.ResponseWriter, r *http.Request) {
	out := &argOutGetFeatureList{
		FeatureList: soap.XMLLite(
			fmt.Sprintf(`<Features xmlns="urn:schemas-upnp-org:av:avs" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">
	<Feature name="samsung.com_BASICVIEW" version="1">
		<container id="%s" type="object.item.audioItem"/>
		<container id="%s" type="object.item.videoItem"/>
		<container id="%s" type="object.item.imageItem"/>
	</Feature>
</Features>`, storage.MusicID, storage.VideoID, storage.ImageID)),
	}
	soap.SendActionResponse(soapAction, out, w)
}
