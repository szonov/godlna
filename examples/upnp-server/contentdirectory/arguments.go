package contentdirectory

type ArgInGetSearchCapabilities struct {
}
type ArgOutGetSearchCapabilities struct {
	SearchCaps string `scpd:"SearchCapabilities,string"`
}
type ArgInGetSortCapabilities struct {
}
type ArgOutGetSortCapabilities struct {
	SortCaps string `scpd:"SortCapabilities,string"`
}
type ArgInGetSystemUpdateID struct {
}
type ArgOutGetSystemUpdateID struct {
	Id uint32 `scpd:"SystemUpdateID,ui4,events"`
}
type ArgInBrowse struct {
	ObjectID       string `scpd:"A_ARG_TYPE_ObjectID,string"`
	BrowseFlag     string `scpd:"A_ARG_TYPE_BrowseFlag,string BrowseMetadata,BrowseDirectChildren"`
	Filter         string `scpd:"A_ARG_TYPE_Filter,string"`
	StartingIndex  uint32 `scpd:"A_ARG_TYPE_Index,ui4"`
	RequestedCount uint32 `scpd:"A_ARG_TYPE_Count,ui4"`
	SortCriteria   string `scpd:"A_ARG_TYPE_SortCriteria,string"`
}
type ArgOutBrowse struct {
	Result         string `scpd:"A_ARG_TYPE_Result,string"`
	NumberReturned uint32 `scpd:"A_ARG_TYPE_Count,ui4"`
	TotalMatches   uint32 `scpd:"A_ARG_TYPE_Count,ui4"`
	UpdateID       uint32 `scpd:"A_ARG_TYPE_UpdateID,ui4"`
}
type ArgInSearch struct {
	ContainerID    string `scpd:"A_ARG_TYPE_ObjectID,string"`
	SearchCriteria string `scpd:"A_ARG_TYPE_SearchCriteria,string"`
	Filter         string `scpd:"A_ARG_TYPE_Filter,string"`
	StartingIndex  uint32 `scpd:"A_ARG_TYPE_Index,ui4"`
	RequestedCount uint32 `scpd:"A_ARG_TYPE_Count,ui4"`
	SortCriteria   string `scpd:"A_ARG_TYPE_SortCriteria,string"`
}
type ArgOutSearch struct {
	Result         string `scpd:"A_ARG_TYPE_Result,string"`
	NumberReturned uint32 `scpd:"A_ARG_TYPE_Count,ui4"`
	TotalMatches   uint32 `scpd:"A_ARG_TYPE_Count,ui4"`
	UpdateID       uint32 `scpd:"A_ARG_TYPE_UpdateID,ui4"`
}
type ArgInCreateObject struct {
	ContainerID string `scpd:"A_ARG_TYPE_ObjectID,string"`
	Elements    string `scpd:"A_ARG_TYPE_Result,string"`
}
type ArgOutCreateObject struct {
	ObjectID string `scpd:"A_ARG_TYPE_ObjectID,string"`
	Result   string `scpd:"A_ARG_TYPE_Result,string"`
}
type ArgInDestroyObject struct {
	ObjectID string `scpd:"A_ARG_TYPE_ObjectID,string"`
}
type ArgOutDestroyObject struct {
}
type ArgInUpdateObject struct {
	ObjectID        string `scpd:"A_ARG_TYPE_ObjectID,string"`
	CurrentTagValue string `scpd:"A_ARG_TYPE_TagValueList,string"`
	NewTagValue     string `scpd:"A_ARG_TYPE_TagValueList,string"`
}
type ArgOutUpdateObject struct {
}
type ArgInImportResource struct {
	SourceURI      string `scpd:"A_ARG_TYPE_URI,uri"`
	DestinationURI string `scpd:"A_ARG_TYPE_URI,uri"`
}
type ArgOutImportResource struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID,ui4"`
}
type ArgInExportResource struct {
	SourceURI      string `scpd:"A_ARG_TYPE_URI,uri"`
	DestinationURI string `scpd:"A_ARG_TYPE_URI,uri"`
}
type ArgOutExportResource struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID,ui4"`
}
type ArgInStopTransferResource struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID,ui4"`
}
type ArgOutStopTransferResource struct {
}
type ArgInGetTransferProgress struct {
	TransferID uint32 `scpd:"A_ARG_TYPE_TransferID,ui4"`
}
type ArgOutGetTransferProgress struct {
	TransferStatus string `scpd:"A_ARG_TYPE_TransferStatus,string COMPLETED,ERROR,IN_PROGRESS,STOPPED"`
	TransferLength string `scpd:"A_ARG_TYPE_TransferLength,string"`
	TransferTotal  string `scpd:"A_ARG_TYPE_TransferTotal,string"`
}
type ArgInDeleteResource struct {
	ResourceURI string `scpd:"A_ARG_TYPE_URI,uri"`
}
type ArgOutDeleteResource struct {
}
type ArgInCreateReference struct {
	ContainerID string `scpd:"A_ARG_TYPE_ObjectID,string"`
	ObjectID    string `scpd:"A_ARG_TYPE_ObjectID,string"`
}
type ArgOutCreateReference struct {
	NewID string `scpd:"A_ARG_TYPE_ObjectID,string"`
}
