package contentdirectory

import (
	"github.com/szonov/go-upnp-lib/scpd"
)

type ArgInGetSearchCapabilities struct {
}
type ArgOutGetSearchCapabilities struct {
	SearchCaps string
}
type ArgInGetSortCapabilities struct {
}
type ArgOutGetSortCapabilities struct {
	SortCaps string
}
type ArgInGetSystemUpdateID struct {
}
type ArgOutGetSystemUpdateID struct {
	Id scpd.UI4 `events:"yes"`
}
type ArgInBrowse struct {
	ObjectID       string
	BrowseFlag     string `allowed:"BrowseMetadata,BrowseDirectChildren"`
	Filter         string
	StartingIndex  scpd.UI4
	RequestedCount scpd.UI4
	SortCriteria   string
}
type ArgOutBrowse struct {
	Result         string
	NumberReturned scpd.UI4
	TotalMatches   scpd.UI4
	UpdateID       scpd.UI4
}
type ArgInSearch struct {
	ContainerID    string
	SearchCriteria string
	Filter         string
	StartingIndex  scpd.UI4
	RequestedCount scpd.UI4
	SortCriteria   string
}
type ArgOutSearch struct {
	Result         string
	NumberReturned scpd.UI4
	TotalMatches   scpd.UI4
	UpdateID       scpd.UI4
}
type ArgInCreateObject struct {
	ContainerID string
	Elements    string
}
type ArgOutCreateObject struct {
	ObjectID string
	Result   string
}
type ArgInDestroyObject struct {
	ObjectID string
}
type ArgOutDestroyObject struct {
}
type ArgInUpdateObject struct {
	ObjectID        string
	CurrentTagValue string
	NewTagValue     string
}
type ArgOutUpdateObject struct {
}
type ArgInImportResource struct {
	SourceURI      scpd.URI
	DestinationURI scpd.URI
}
type ArgOutImportResource struct {
	TransferID scpd.UI4
}
type ArgInExportResource struct {
	SourceURI      scpd.URI
	DestinationURI scpd.URI
}
type ArgOutExportResource struct {
	TransferID scpd.UI4
}
type ArgInStopTransferResource struct {
	TransferID scpd.UI4
}
type ArgOutStopTransferResource struct {
}
type ArgInGetTransferProgress struct {
	TransferID scpd.UI4
}
type ArgOutGetTransferProgress struct {
	TransferStatus string `allowed:"COMPLETED,ERROR,IN_PROGRESS,STOPPED"`
	TransferLength string
	TransferTotal  string
}
type ArgInDeleteResource struct {
	ResourceURI scpd.URI
}
type ArgOutDeleteResource struct {
}
type ArgInCreateReference struct {
	ContainerID string
	ObjectID    string
}
type ArgOutCreateReference struct {
	NewID string
}
