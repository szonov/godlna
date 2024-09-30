package contentdirectory

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
	Id uint32 `scpd:"SystemUpdateID"`
}
type ArgInBrowse struct {
	ObjectID       string `scpd:"A_ARG_TYPE_ObjectID"`
	BrowseFlag     string `scpd:"A_ARG_TYPE_BrowseFlag"`
	Filter         string `scpd:"A_ARG_TYPE_Filter"`
	StartingIndex  uint32 `scpd:"A_ARG_TYPE_Index"`
	RequestedCount uint32 `scpd:"A_ARG_TYPE_Count"`
	SortCriteria   string `scpd:"A_ARG_TYPE_SortCriteria"`
}
type ArgOutBrowse struct {
	Result         string `scpd:"A_ARG_TYPE_Result"`
	NumberReturned uint32 `scpd:"A_ARG_TYPE_Count"`
	TotalMatches   uint32 `scpd:"A_ARG_TYPE_Count"`
	UpdateID       uint32 `scpd:"A_ARG_TYPE_UpdateID"`
}
type ArgInSearch struct {
	ContainerID    string `scpd:"A_ARG_TYPE_ObjectID"`
	SearchCriteria string `scpd:"A_ARG_TYPE_SearchCriteria"`
	Filter         string `scpd:"A_ARG_TYPE_Filter"`
	StartingIndex  uint32 `scpd:"A_ARG_TYPE_Index"`
	RequestedCount uint32 `scpd:"A_ARG_TYPE_Count"`
	SortCriteria   string `scpd:"A_ARG_TYPE_SortCriteria"`
}
type ArgOutSearch struct {
	Result         string `scpd:"A_ARG_TYPE_Result"`
	NumberReturned uint32 `scpd:"A_ARG_TYPE_Count"`
	TotalMatches   uint32 `scpd:"A_ARG_TYPE_Count"`
	UpdateID       uint32 `scpd:"A_ARG_TYPE_UpdateID"`
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
	SourceURI      string `scpd:"A_ARG_TYPE_URI"`
	DestinationURI string `scpd:"A_ARG_TYPE_URI"`
}
type ArgOutImportResource struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgInExportResource struct {
	SourceURI      string `scpd:"A_ARG_TYPE_URI"`
	DestinationURI string `scpd:"A_ARG_TYPE_URI"`
}
type ArgOutExportResource struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgInStopTransferResource struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgOutStopTransferResource struct {
}
type ArgInGetTransferProgress struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID"`
}
type ArgOutGetTransferProgress struct {
	TransferStatus string `scpd:"A_ARG_TYPE_TransferStatus"`
	TransferLength string `scpd:"A_ARG_TYPE_TransferLength"`
	TransferTotal  string `scpd:"A_ARG_TYPE_TransferTotal"`
}
type ArgInDeleteResource struct {
	ResourceURI string `scpd:"A_ARG_TYPE_URI"`
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
