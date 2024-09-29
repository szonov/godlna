package upnp

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/user"
	"strings"
)

const (
	DefaultDeviceType   = "urn:schemas-upnp-org:device:MediaServer:1"
	DefaultManufacturer = "Private"
	DefaultModelName    = "UPNP Server"
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
	return "UPNP Server"
}
