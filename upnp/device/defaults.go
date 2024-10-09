package device

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/user"
	"strings"
)

const (
	DefaultDeviceType   = "urn:schemas-upnp-org:device:MediaServer:1"
	DefaultManufacturer = "Home"
	DefaultModelName    = "DLNA Server"
)

func NewUDN(unique string) string {
	hash := md5.Sum([]byte(unique))
	return fmt.Sprintf("uuid:%x-%x-%x-%x-%x", hash[:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

func DefaultFriendlyName() string {
	if name, err := os.Hostname(); err == nil {
		pos := strings.Index(name, ".")
		if pos > 0 {
			return name[:pos]
		}
		return name
	}
	if u, err := user.Current(); err == nil {
		return u.Name
	}
	return "DLNA Server"
}

func DefaultDeviceDescription() *Description {

	friendlyName := DefaultFriendlyName()

	return &Description{
		SpecVersion: Version,
		Device: &Device{
			DeviceType:   DefaultDeviceType,
			FriendlyName: friendlyName,
			UDN:          NewUDN(friendlyName),
			Manufacturer: DefaultManufacturer,
			ModelName:    DefaultModelName,
			ServiceList:  make([]*Service, 0),
		},
		Location: "/rootDesc.xml",
	}
}
