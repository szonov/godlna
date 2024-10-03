package contentdirectory

import (
	"github.com/szonov/go-upnp-lib/handler"
)

func (ctl *ServiceController) createHandler() *ServiceController {
	ctl.Handler = &handler.Handler{
		ServiceType: ctl.Service.ServiceType,
		Actions: handler.ActionMap{
			"GetSearchCapabilities": func() *handler.Action {
				return handler.NewAction(ctl.GetSearchCapabilities, &ArgInGetSearchCapabilities{}, &ArgOutGetSearchCapabilities{})
			},
			"GetSortCapabilities": func() *handler.Action {
				return handler.NewAction(ctl.GetSortCapabilities, &ArgInGetSortCapabilities{}, &ArgOutGetSortCapabilities{})
			},
			"GetSystemUpdateID": func() *handler.Action {
				return handler.NewAction(ctl.GetSystemUpdateID, &ArgInGetSystemUpdateID{}, &ArgOutGetSystemUpdateID{})
			},
			"Browse": func() *handler.Action {
				return handler.NewAction(ctl.Browse, &ArgInBrowse{}, &ArgOutBrowse{})
			},
			"Search": func() *handler.Action {
				return handler.NewAction(ctl.Search, &ArgInSearch{}, &ArgOutSearch{})
			},
			"CreateObject": func() *handler.Action {
				return handler.NewAction(ctl.CreateObject, &ArgInCreateObject{}, &ArgOutCreateObject{})
			},
			"DestroyObject": func() *handler.Action {
				return handler.NewAction(ctl.DestroyObject, &ArgInDestroyObject{}, &ArgOutDestroyObject{})
			},
			"UpdateObject": func() *handler.Action {
				return handler.NewAction(ctl.UpdateObject, &ArgInUpdateObject{}, &ArgOutUpdateObject{})
			},
			"ImportResource": func() *handler.Action {
				return handler.NewAction(ctl.ImportResource, &ArgInImportResource{}, &ArgOutImportResource{})
			},
			"ExportResource": func() *handler.Action {
				return handler.NewAction(ctl.ExportResource, &ArgInExportResource{}, &ArgOutExportResource{})
			},
			"StopTransferResource": func() *handler.Action {
				return handler.NewAction(ctl.StopTransferResource, &ArgInStopTransferResource{}, &ArgOutStopTransferResource{})
			},
			"GetTransferProgress": func() *handler.Action {
				return handler.NewAction(ctl.GetTransferProgress, &ArgInGetTransferProgress{}, &ArgOutGetTransferProgress{})
			},
			"DeleteResource": func() *handler.Action {
				return handler.NewAction(ctl.DeleteResource, &ArgInDeleteResource{}, &ArgOutDeleteResource{})
			},
			"CreateReference": func() *handler.Action {
				return handler.NewAction(ctl.CreateReference, &ArgInCreateReference{}, &ArgOutCreateReference{})
			},
		},
	}
	return ctl
}
