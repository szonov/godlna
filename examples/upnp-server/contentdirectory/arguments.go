package contentdirectory

import (
	"github.com/szonov/go-upnp-lib/scpd"
)

type ArgInGetSearchCapabilities struct {
}
type ArgOutGetSearchCapabilities struct {
	SearchCaps string `scpd:"SearchCapabilities"`
}
type ArgInGetSortCapabilities struct {
}
type ArgOutGetSortCapabilities struct {
	SortCaps string `scpd:"SortCapabilities"`
}
type ArgInGetSystemUpdateID struct {
}
type ArgOutGetSystemUpdateID struct {
	Id scpd.UI4 `scpd:"SystemUpdateID" events:"yes"`
}
type ArgInBrowse struct {
	ObjectID       string   `scpd:"A_ARG_TYPE_ObjectID"`
	BrowseFlag     string   `scpd:"A_ARG_TYPE_BrowseFlag" allowed:"BrowseMetadata,BrowseDirectChildren"`
	Filter         string   `scpd:"A_ARG_TYPE_Filter"`
	StartingIndex  scpd.UI4 `scpd:"A_ARG_TYPE_Index"`
	RequestedCount scpd.UI4 `scpd:"A_ARG_TYPE_Count"`
	SortCriteria   string   `scpd:"A_ARG_TYPE_SortCriteria"`
}
type ArgOutBrowse struct {
	Result         string   `scpd:"A_ARG_TYPE_Result"`
	NumberReturned scpd.UI4 `scpd:"A_ARG_TYPE_Count"`
	TotalMatches   scpd.UI4 `scpd:"A_ARG_TYPE_Count"`
	UpdateID       scpd.UI4 `scpd:"A_ARG_TYPE_UpdateID"`
}
type ArgInSearch struct {
	ContainerID    string   `scpd:"A_ARG_TYPE_ObjectID"`
	SearchCriteria string   `scpd:"A_ARG_TYPE_SearchCriteria"`
	Filter         string   `scpd:"A_ARG_TYPE_Filter"`
	StartingIndex  scpd.UI4 `scpd:"A_ARG_TYPE_Index"`
	RequestedCount scpd.UI4 `scpd:"A_ARG_TYPE_Count"`
	SortCriteria   string   `scpd:"A_ARG_TYPE_SortCriteria"`
}
type ArgOutSearch struct {
	Result         string   `scpd:"A_ARG_TYPE_Result"`
	NumberReturned scpd.UI4 `scpd:"A_ARG_TYPE_Count"`
	TotalMatches   scpd.UI4 `scpd:"A_ARG_TYPE_Count"`
	UpdateID       scpd.UI4 `scpd:"A_ARG_TYPE_UpdateID"`
}
type ArgInCreateObject struct {
	ContainerID string `scpd:"A_ARG_TYPE_ObjectID"`
	Elements    string `scpd:"A_ARG_TYPE_Result"`
}
type ArgOutCreateObject struct {
	ObjectID string `scpd:"A_ARG_TYPE_ObjectID"`
	Result   string `scpd:"A_ARG_TYPE_Result"`
}
type ArgInDestroyObject struct {
	ObjectID string `scpd:"A_ARG_TYPE_ObjectID"`
}
type ArgOutDestroyObject struct {
}
type ArgInUpdateObject struct {
	ObjectID        string `scpd:"A_ARG_TYPE_ObjectID"`
	CurrentTagValue string `scpd:"A_ARG_TYPE_TagValueList"`
	NewTagValue     string `scpd:"A_ARG_TYPE_TagValueList"`
}
type ArgOutUpdateObject struct {
}
type ArgInImportResource struct {
	SourceURI      scpd.URI `scpd:"A_ARG_TYPE_URI"`
	DestinationURI scpd.URI `scpd:"A_ARG_TYPE_URI"`
}
type ArgOutImportResource struct {
	TransferID scpd.UI4 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgInExportResource struct {
	SourceURI      scpd.URI `scpd:"A_ARG_TYPE_URI"`
	DestinationURI scpd.URI `scpd:"A_ARG_TYPE_URI"`
}
type ArgOutExportResource struct {
	TransferID scpd.UI4 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgInStopTransferResource struct {
	TransferID scpd.UI4 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgOutStopTransferResource struct {
}
type ArgInGetTransferProgress struct {
	TransferID scpd.UI4 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgOutGetTransferProgress struct {
	TransferStatus string `scpd:"A_ARG_TYPE_TransferStatus" allowed:"COMPLETED,ERROR,IN_PROGRESS,STOPPED"`
	TransferLength string `scpd:"A_ARG_TYPE_TransferLength"`
	TransferTotal  string `scpd:"A_ARG_TYPE_TransferTotal"`
}
type ArgInDeleteResource struct {
	ResourceURI scpd.URI `scpd:"A_ARG_TYPE_URI"`
}
type ArgOutDeleteResource struct {
}
type ArgInCreateReference struct {
	ContainerID string `scpd:"A_ARG_TYPE_ObjectID"`
	ObjectID    string `scpd:"A_ARG_TYPE_ObjectID"`
}
type ArgOutCreateReference struct {
	NewID string `scpd:"A_ARG_TYPE_ObjectID"`
}
