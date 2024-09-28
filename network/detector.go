package network

import (
	"net"
)

type V4Interface struct {
	Interface *net.Interface
	IP        string
}

func (v V4Interface) Valid() bool {
	return v.Interface != nil
}

func (v V4Interface) String() string {
	if v.Valid() {
		return v.IP + " (" + v.Interface.Name + ")"
	}
	return "-.-.-.- (-)"
}

// DefaultV4Interface Find default interface and default ip (v4) on this interface
// filters:
// - first  = name of interface, empty = any
// - second = ip,                empty = any
func DefaultV4Interface(filters ...string) V4Interface {

	ifaceFilter := ""
	ipFilter := ""

	if len(filters) > 0 {
		ifaceFilter = filters[0]
	}

	if len(filters) > 1 {
		ipFilter = filters[1]
	}

	ret := V4Interface{}

	if ifaces, err := net.Interfaces(); err == nil {
		for _, address := range ifaces {

			if ifaceFilter != "" && ifaceFilter != address.Name {
				continue
			}

			if address.Flags&net.FlagUp == 0 || address.MTU <= 0 {
				continue
			}

			if addrs, err := address.Addrs(); err == nil {
				for _, addr := range addrs {
					if ip := getIpV4(addr, ipFilter); ip != nil {
						ret.Interface = &address
						ret.IP = *ip
						return ret
					}
				}
			}
		}
	}

	return ret
}

func getIpV4(addr net.Addr, ipFilter string) *string {
	ipnet, ok := addr.(*net.IPNet)
	if !ok {
		return nil
	}

	// in auto detection skip 127.0.0.1 address
	if ipFilter == "" && ipnet.IP.IsLoopback() {
		return nil
	}

	// skip non v4 ip
	if ipnet.IP.To4() == nil {
		return nil
	}

	ip := ipnet.IP.String()

	// if configured auto-detection or found IP equal to configured IP - we are success
	if ipFilter == "" || ipFilter == ip {
		return &ip
	}

	return nil
}
