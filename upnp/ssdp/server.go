package ssdp

const (
	MulticastAddrPort = "239.255.255.250:1900"
	Alive             = "ssdp:alive"
	Bye               = "ssdp:byebye"
)

type Server interface {
	Start() error
	Stop()
}
