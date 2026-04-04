//go:build darwin

package fswatcher

// FSEvents flag constants, mirroring C definitions
const (
	FSEventStreamEventFlagNone               uint32 = 0x00000000
	FSEventStreamEventFlagMustScanSubDirs    uint32 = 0x00000001
	FSEventStreamEventFlagUserDropped        uint32 = 0x00000002
	FSEventStreamEventFlagKernelDropped      uint32 = 0x00000004
	FSEventStreamEventFlagEventIdsWrapped    uint32 = 0x00000008
	FSEventStreamEventFlagHistoryDone        uint32 = 0x00000010
	FSEventStreamEventFlagRootChanged        uint32 = 0x00000020
	FSEventStreamEventFlagMount              uint32 = 0x00000040
	FSEventStreamEventFlagUnmount            uint32 = 0x00000080
	FSEventStreamEventFlagOwnEvent           uint32 = 0x00080000
	FSEventStreamEventFlagItemCreated        uint32 = 0x00000100
	FSEventStreamEventFlagItemRemoved        uint32 = 0x00000200
	FSEventStreamEventFlagItemInodeMetaMod   uint32 = 0x00000400
	FSEventStreamEventFlagItemRenamed        uint32 = 0x00000800
	FSEventStreamEventFlagItemModified       uint32 = 0x00001000
	FSEventStreamEventFlagItemFinderInfoMod  uint32 = 0x00002000
	FSEventStreamEventFlagItemChangeOwner    uint32 = 0x00004000
	FSEventStreamEventFlagItemXattrMod       uint32 = 0x00008000
	FSEventStreamEventFlagItemIsFile         uint32 = 0x00010000
	FSEventStreamEventFlagItemIsDir          uint32 = 0x00020000
	FSEventStreamEventFlagItemIsSymlink      uint32 = 0x00040000
	FSEventStreamEventFlagItemIsHardlink     uint32 = 0x00100000
	FSEventStreamEventFlagItemIsLastHardlink uint32 = 0x00200000
	FSEventStreamEventFlagItemCloned         uint32 = 0x00400000
)

// flagInfo holds the value and name for an FSEvent flag
type flagInfo struct {
	value uint32
	name  string
}

// knownFlags maps FSEvent flag values to their string names
var knownFlags = []flagInfo{
	{FSEventStreamEventFlagMustScanSubDirs, "MustScanSubDirs"},
	{FSEventStreamEventFlagUserDropped, "UserDropped"},
	{FSEventStreamEventFlagKernelDropped, "KernelDropped"},
	{FSEventStreamEventFlagEventIdsWrapped, "EventIdsWrapped"},
	{FSEventStreamEventFlagHistoryDone, "HistoryDone"},
	{FSEventStreamEventFlagRootChanged, "RootChanged"},
	{FSEventStreamEventFlagMount, "Mount"},
	{FSEventStreamEventFlagUnmount, "Unmount"},
	{FSEventStreamEventFlagOwnEvent, "OwnEvent"},
	{FSEventStreamEventFlagItemIsFile, "IsFile"},
	{FSEventStreamEventFlagItemIsDir, "IsDir"},
	{FSEventStreamEventFlagItemIsSymlink, "IsSymlink"},
	{FSEventStreamEventFlagItemIsHardlink, "IsHardlink"},
	{FSEventStreamEventFlagItemIsLastHardlink, "IsLastHardlink"},
	{FSEventStreamEventFlagItemCreated, "Created"},
	{FSEventStreamEventFlagItemRemoved, "Removed"},
	{FSEventStreamEventFlagItemRenamed, "Renamed"},
	{FSEventStreamEventFlagItemModified, "Modified"},
	{FSEventStreamEventFlagItemInodeMetaMod, "InodeMetaMod"},
	{FSEventStreamEventFlagItemFinderInfoMod, "FinderInfoMod"},
	{FSEventStreamEventFlagItemChangeOwner, "ChangeOwner"},
	{FSEventStreamEventFlagItemXattrMod, "XattrMod"},
	{FSEventStreamEventFlagItemCloned, "Cloned"},
}

// ParseDarwinEventFlags converts raw FSEvent flags into a slice of human-readable strings
func ParseDarwinEventFlags(flags uint32) []string {
	if flags == FSEventStreamEventFlagNone {
		return nil
	}
	var descriptions []string
	for _, info := range knownFlags {
		if flags&info.value != 0 {
			descriptions = append(descriptions, info.name)
		}
	}
	return descriptions
}
