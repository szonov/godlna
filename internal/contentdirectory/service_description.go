package contentdirectory

import (
	"github.com/szonov/godlna/pkg/upnp/scpd"
)

func NewServiceDescription() *scpd.Document {
	return scpd.NewDocumentBuilder().
		Version(1, 0).
		Action("GetSearchCapabilities",
			scpd.OUT("SearchCaps", "SearchCapabilities"),
		).
		Action("GetSortCapabilities",
			scpd.OUT("SortCaps", "SortCapabilities"),
		).
		Action("GetSystemUpdateID",
			scpd.OUT("Id", "SystemUpdateID"),
		).
		Action("Browse",
			scpd.IN("ObjectID", "A_ARG_TYPE_ObjectID"),
			scpd.IN("BrowseFlag", "A_ARG_TYPE_BrowseFlag"),
			scpd.IN("Filter", "A_ARG_TYPE_Filter"),
			scpd.IN("StartingIndex", "A_ARG_TYPE_Index"),
			scpd.IN("RequestedCount", "A_ARG_TYPE_Count"),
			scpd.IN("SortCriteria", "A_ARG_TYPE_SortCriteria"),
			scpd.OUT("Result", "A_ARG_TYPE_Result"),
			scpd.OUT("NumberReturned", "A_ARG_TYPE_Count"),
			scpd.OUT("TotalMatches", "A_ARG_TYPE_Count"),
			scpd.OUT("UpdateID", "A_ARG_TYPE_UpdateID"),
		).
		//Action("Search",
		//	scpd.IN("ContainerID", "A_ARG_TYPE_ObjectID"),
		//	scpd.IN("SearchCriteria", "A_ARG_TYPE_SearchCriteria"),
		//	scpd.IN("Filter", "A_ARG_TYPE_Filter"),
		//	scpd.IN("StartingIndex", "A_ARG_TYPE_Index"),
		//	scpd.IN("RequestedCount", "A_ARG_TYPE_Count"),
		//	scpd.IN("SortCriteria", "A_ARG_TYPE_SortCriteria"),
		//	scpd.OUT("Result", "A_ARG_TYPE_Result"),
		//	scpd.OUT("NumberReturned", "A_ARG_TYPE_Count"),
		//	scpd.OUT("TotalMatches", "A_ARG_TYPE_Count"),
		//	scpd.OUT("UpdateID", "A_ARG_TYPE_UpdateID"),
		//).
		//Action("CreateObject",
		//	scpd.IN("ContainerID", "A_ARG_TYPE_ObjectID"),
		//	scpd.IN("Elements", "A_ARG_TYPE_Result"),
		//	scpd.OUT("ObjectID", "A_ARG_TYPE_ObjectID"),
		//	scpd.OUT("Result", "A_ARG_TYPE_Result"),
		//).
		//Action("DestroyObject",
		//	scpd.IN("ObjectID", "A_ARG_TYPE_ObjectID"),
		//).
		//Action("UpdateObject",
		//	scpd.IN("ObjectID", "A_ARG_TYPE_ObjectID"),
		//	scpd.IN("CurrentTagValue", "A_ARG_TYPE_TagValueList"),
		//	scpd.IN("NewTagValue", "A_ARG_TYPE_TagValueList"),
		//).
		//Action("ImportResource",
		//	scpd.IN("SourceURI", "A_ARG_TYPE_URI"),
		//	scpd.IN("DestinationURI", "A_ARG_TYPE_URI"),
		//	scpd.OUT("TransferID", "A_ARG_TYPE_TransferID"),
		//).
		//Action("ExportResource",
		//	scpd.IN("SourceURI", "A_ARG_TYPE_URI"),
		//	scpd.IN("DestinationURI", "A_ARG_TYPE_URI"),
		//	scpd.OUT("TransferID", "A_ARG_TYPE_TransferID"),
		//).
		//Action("StopTransferResource",
		//	scpd.IN("TransferID", "A_ARG_TYPE_TransferID"),
		//).
		//Action("GetTransferProgress",
		//	scpd.IN("TransferID", "A_ARG_TYPE_TransferID"),
		//	scpd.OUT("TransferStatus", "A_ARG_TYPE_TransferStatus"),
		//	scpd.OUT("TransferLength", "A_ARG_TYPE_TransferLength"),
		//	scpd.OUT("TransferTotal", "A_ARG_TYPE_TransferTotal"),
		//).
		//Action("DeleteResource",
		//	scpd.IN("ResourceURI", "A_ARG_TYPE_URI"),
		//).
		//Action("CreateReference",
		//	scpd.IN("ContainerID", "A_ARG_TYPE_ObjectID"),
		//	scpd.IN("ObjectID", "A_ARG_TYPE_ObjectID"),
		//	scpd.OUT("NewID", "A_ARG_TYPE_ObjectID"),
		//).
		Action("X_GetFeatureList",
			scpd.OUT("FeatureList", "X_ARG_TYPE_FeatureList"),
		).
		Action("X_SetBookmark",
			scpd.IN("CategoryType", "X_ARG_TYPE_CategoryType"),
			scpd.IN("RID", "X_ARG_TYPE_RID"),
			scpd.IN("ObjectID", "A_ARG_TYPE_ObjectID"),
			scpd.IN("PosSecond", "X_ARG_TYPE_PosSec"),
		).
		//Variable("TransferIDs", "string", scpd.Events()).
		Variable("A_ARG_TYPE_ObjectID", "string").
		Variable("A_ARG_TYPE_Result", "string").
		//Variable("A_ARG_TYPE_SearchCriteria", "string").
		Variable("A_ARG_TYPE_BrowseFlag", "string",
			scpd.Only("BrowseMetadata", "BrowseDirectChildren"),
		).
		Variable("A_ARG_TYPE_Filter", "string").
		Variable("A_ARG_TYPE_SortCriteria", "string").
		Variable("A_ARG_TYPE_Index", "ui4").
		Variable("A_ARG_TYPE_Count", "ui4").
		Variable("A_ARG_TYPE_UpdateID", "ui4").
		//Variable("A_ARG_TYPE_TransferID", "ui4").
		//Variable("A_ARG_TYPE_TransferStatus", "string",
		//	scpd.Only("COMPLETED", "ERROR", "IN_PROGRESS", "STOPPED"),
		//).
		//Variable("A_ARG_TYPE_TransferLength", "string").
		//Variable("A_ARG_TYPE_TransferTotal", "string").
		//Variable("A_ARG_TYPE_TagValueList", "string").
		//Variable("A_ARG_TYPE_URI", "uri").
		Variable("SearchCapabilities", "string").
		Variable("SortCapabilities", "string").
		Variable("SystemUpdateID", "ui4", scpd.Events()).
		Variable("ContainerUpdateIDs", "string", scpd.Events()).
		Variable("X_ARG_TYPE_FeatureList", "string").
		Variable("X_ARG_TYPE_CategoryType", "ui4").
		Variable("X_ARG_TYPE_RID", "ui4").
		Variable("X_ARG_TYPE_PosSec", "ui4").
		Document()
}
