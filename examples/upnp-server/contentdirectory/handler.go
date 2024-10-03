package contentdirectory

import "github.com/szonov/go-upnp-lib/handler"

func (ctl *ServiceController) createHandler() *ServiceController {
	ctl.Handler = &handler.Handler{
		ServiceType: ctl.Service.ServiceType,
		Actions: handler.ActionMap{
			"Browse": func() *handler.Action {
				return handler.NewAction(ctl.Browse, &ArgInBrowse{}, &ArgOutBrowse{})
			},
		},
	}
	return ctl
}
