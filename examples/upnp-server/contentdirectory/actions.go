package contentdirectory

import (
	"github.com/szonov/go-upnp-lib/handler"
)

func (ctl *ServiceController) createActions() []handler.Action {
	return []handler.Action{
		{
			Name: "GetSearchCapabilities",
			Func: ctl.GetSearchCapabilities,
			Args: func() (interface{}, interface{}) {
				return &ArgInGetSearchCapabilities{}, &ArgOutGetSearchCapabilities{}
			},
		},
		{
			Name: "GetSortCapabilities",
			Func: ctl.GetSortCapabilities,
			Args: func() (interface{}, interface{}) {
				return &ArgInGetSortCapabilities{}, &ArgOutGetSortCapabilities{}
			},
		},
		{
			Name: "GetSystemUpdateID",
			Func: ctl.GetSystemUpdateID,
			Args: func() (interface{}, interface{}) {
				return &ArgInGetSystemUpdateID{}, &ArgOutGetSystemUpdateID{}
			},
		},
		{
			Name: "Browse",
			Func: ctl.Browse,
			Args: func() (interface{}, interface{}) {
				return &ArgInBrowse{}, &ArgOutBrowse{}
			},
		},
		{
			Name: "Search",
			Func: ctl.Search,
			Args: func() (interface{}, interface{}) {
				return &ArgInSearch{}, &ArgOutSearch{}
			},
		},
		{
			Name: "CreateObject",
			Func: ctl.CreateObject,
			Args: func() (interface{}, interface{}) {
				return &ArgInCreateObject{}, &ArgOutCreateObject{}
			},
		},
		{
			Name: "DestroyObject",
			Func: ctl.DestroyObject,
			Args: func() (interface{}, interface{}) {
				return &ArgInDestroyObject{}, &ArgOutDestroyObject{}
			},
		},
		{
			Name: "UpdateObject",
			Func: ctl.UpdateObject,
			Args: func() (interface{}, interface{}) {
				return &ArgInUpdateObject{}, &ArgOutUpdateObject{}
			},
		},
		{
			Name: "ImportResource",
			Func: ctl.ImportResource,
			Args: func() (interface{}, interface{}) {
				return &ArgInImportResource{}, &ArgOutImportResource{}
			},
		},
		{
			Name: "ExportResource",
			Func: ctl.ExportResource,
			Args: func() (interface{}, interface{}) {
				return &ArgInExportResource{}, &ArgOutExportResource{}
			},
		},
		{
			Name: "StopTransferResource",
			Func: ctl.StopTransferResource,
			Args: func() (interface{}, interface{}) {
				return &ArgInStopTransferResource{}, &ArgOutStopTransferResource{}
			},
		},
		{
			Name: "GetTransferProgress",
			Func: ctl.GetTransferProgress,
			Args: func() (interface{}, interface{}) {
				return &ArgInGetTransferProgress{}, &ArgOutGetTransferProgress{}
			},
		},
		{
			Name: "DeleteResource",
			Func: ctl.DeleteResource,
			Args: func() (interface{}, interface{}) {
				return &ArgInDeleteResource{}, &ArgOutDeleteResource{}
			},
		},
		{
			Name: "CreateReference",
			Func: ctl.CreateReference,
			Args: func() (interface{}, interface{}) {
				return &ArgInCreateReference{}, &ArgOutCreateReference{}
			},
		},
	}
}
