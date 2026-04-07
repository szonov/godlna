package ssdp

const (
	MulticastAddrPort = "239.255.255.250:1900"
	rootDevice        = "upnp:rootdevice"
	Alive             = "ssdp:alive"
	Bye               = "ssdp:byebye"
)

type Server interface {
	Start() error
	Stop() error
}
