package contentdirectory

import "github.com/szonov/go-upnp-lib/scpd"

func (ctl *Controller) createHandler() *Controller {
	ctl.Handler = &scpd.Handler{
		ServiceType: ctl.Service.ServiceType,
		Actions: scpd.HandlerActionMap{
			"Browse": func() *scpd.HandlerAction {
				return scpd.NewHandlerAction(ctl.Browse, &ArgInBrowse{}, &ArgOutBrowse{})
			},
			// "Browse": {Func: me.Browse, In: &ArgInBrowse{}, Out: &ArgOutBrowse{}},
		},
	}
	return ctl
}
