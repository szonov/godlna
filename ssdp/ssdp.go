package ssdp

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	MulticastAddrPort string = "239.255.255.250:1900"
	CallerName        string = "ssdp"
)

// Server Describes structure of SSDP Server
// reference: https://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.0-20080424.pdf
type Server struct {
	// [page 16] LOCATION Required.
	// Contains a URL to the UPnP description of the root device. Normally the host portion contains a literal IP
	// address rather than a domain name in unmanaged networks. Specified by UPnP vendor. Single URL.
	// Example: http://192.168.0.100/rootDesc.xml
	// Required: No defaults
	Location string

	// [page 16] SERVER Required. Concatenation of OS name, OS version, UPnP/1.0, product name, and product version.
	// Specified by UPnP vendor. String.
	// Must accurately reflect the version number of the UPnP Device Architecture supported by the device.
	// Control points must be prepared to accept a higher minor version number than the control point itself implements.
	// For example, control points implementing UDA version 1.0 will be able to interoperate with devices
	// implementing UDA version 1.1.
	// Example: "Linux/6.0 UPnP/1.0 App/1.0"
	// Optional: Default is "[runtime.GOOS]/[runtime.Version()] UPnP/1.0 GoUPnP/1.0"
	ServerHeader string

	// [page 16] CACHE-CONTROL Required. Must have max-age directive that specifies number of seconds the advertisement is valid.
	// After this duration, control points should assume the device (or service) is no longer available.
	// Should be greater than or equal to 1800 seconds (30 minutes). Specified by UPnP vendor. Integer.
	// Optional: Default is 30 minutes
	MaxAge time.Duration

	// [page 15]  In addition, the device must re-send its advertisements periodically prior to expiration
	// of the duration specified in the CACHE-CONTROL header;
	// it is recommended that such refreshing of advertisements be done at a randomly-distributed interval
	// of less than one-half of the advertisement expiration time
	// Optional: Default is "2/5 * MaxAge"
	NotifyInterval time.Duration

	// [page 12] To limit network congestion, the time-to-live (TTL) of each
	// IP packet for each multicast message should default to 4 and should be configurable.
	// Optional: Default is 4
	MulticastTTL int

	// Full device type. Should contain exact value, added to xml in <deviceType>.*</deviceType>
	// Example: "urn:schemas-upnp-org:device:MediaServer:1"
	// Required: No defaults
	DeviceType string

	// Device UUID specified by UPnP vendor. (with or without "uuid:" prefix)
	// Valid examples:
	// - "uuid:da2cc462-0000-0000-0000-44fd2452e03f"
	// - "da2cc462-0000-0000-0000-44fd2452e03f"
	// Required: No defaults
	DeviceUUID string

	// List of full service types as it appears in xml in <serviceType>.*</serviceType>
	// Example: []string{
	//   "urn:schemas-upnp-org:service:ContentDirectory:1",
	//   "urn:schemas-upnp-org:service:ConnectionManager:1",
	// }
	// Required: Not empty. No defaults
	ServiceList []string

	// Used network interface for ssdp server
	// I don't need multi-interface support for my purposes
	// Required: No defaults
	Interface *net.Interface

	// How to handle errors, useful for logs or something else.
	// Optional: No defaults
	ErrorHandler func(err error, caller string)

	// How to handle ssdp server notifications, useful for debug.
	// Optional: No defaults
	InfoHandler func(msg string, caller string)

	// all handled notification type(nt) / search target(st)
	// len = 3 of device + len(ServiceList)
	targets []string

	quit    chan struct{}
	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
}

func (s *Server) Start() *Server {
	go func() {
		if err := s.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	return s
}

func (s *Server) ListenAndServe() error {
	var err error
	if err = s.validateAndSetDefaults(); err != nil {
		return s.notifyError(err)
	}

	msg := fmt.Sprintf("starting ssdp server on address %s (%s)", MulticastAddrPort, s.Interface.Name)
	s.notifyInfo(msg)

	if s.udpAddr, err = net.ResolveUDPAddr("udp4", MulticastAddrPort); err != nil {
		return s.notifyError(err)
	}

	s.udpConn, err = net.ListenMulticastUDP("udp", s.Interface, s.udpAddr)
	if err != nil {
		return s.notifyError(err)
	}

	if err = ipv4.NewPacketConn(s.udpConn).SetMulticastTTL(s.MulticastTTL); err != nil {
		return s.notifyError(err)
	}

	s.quit = make(chan struct{})
	s.listen()

	return nil
}

func (s *Server) Shutdown() {
	close(s.quit)
	if err := s.udpConn.Close(); err != nil {
		_ = s.notifyError(err)
	}
}

// listen Listen for incoming UDP messages
func (s *Server) listen() {
	for {
		size := s.Interface.MTU
		if size <= 0 || size > 65536 {
			size = 65536
		}
		b := make([]byte, size)
		n, addr, err := s.udpConn.ReadFromUDP(b)
		select {
		case <-s.quit:
			return
		default:
		}
		if s.notifyError(err) != nil {
			break
		}
		go s.parseUdpMessage(b[:n], addr)
	}
}

func (s *Server) parseUdpMessage(buf []byte, sender *net.UDPAddr) {

	lines := strings.Split(string(buf), "\r\n")

	if len(lines) < 5 {
		return
	}

	var hostOK, manOK bool
	mx := int64(1)
	st := ""

	// If value not in our POI (point of interests), we just stop processing and return
	for i, line := range lines {
		if i == 0 {
			// [page 18] request should be started exact by this line
			if line != "M-SEARCH * HTTP/1.1" {
				return
			}
			continue
		}

		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {

			name := strings.ToUpper(strings.Trim(parts[0], " \t"))
			value := strings.Trim(parts[1], " \t\"")

			switch name {
			case "HOST":
				// [page 19] Multicast channel and port reserved for SSDP by
				// Internet Assigned Numbers Authority (IANA). Must be
				// 239.255.255.250:1900. If the port number (“:1900”) is omitted,
				// the receiver should assume the default SSDP port number of 1900.
				if value != MulticastAddrPort && value+":1900" != MulticastAddrPort {
					return
				}
				hostOK = true
			case "MAN":
				// [page 19] Required by HTTP Extension Framework. Unlike the NTS and ST headers,
				// the value of the MAN header is enclosed in double quotes;
				// it defines the scope (namespace) of the extension. Must be "ssdp:discover".
				if value != "ssdp:discover" {
					return
				}
				manOK = true
			case "MX":
				// [page 19] Required. Maximum wait time in seconds.
				// Should be between 1 and 120 inclusive. Device responses should be delayed a
				// random duration between 0 and this many seconds to balance load
				// for the control point when it processes responses.
				mxUint, err := strconv.ParseUint(value, 0, 0)
				if s.notifyError(err) != nil {
					return
				}
				mx = int64(mxUint)
				if mx <= 0 {
					mx = 1
				} else if mx > 120 {
					mx = 120
				}
			case "ST":
				st = value
			}
		}
	}

	if !hostOK || !manOK || st == "" {
		return
	}

	targets := func(st string) []string {
		if st == "ssdp:all" {
			return s.targets
		}
		for _, t := range s.targets {
			if t == st {
				return []string{t}
			}
		}
		return nil
	}(st)

	if len(targets) == 0 {
		return
	}

	s.notifyInfo(fmt.Sprintf("ssdp:discover [%s] from %s", st, sender.String()))
}

func (s *Server) validateAndSetDefaults() error {
	if s.Location == "" {
		return fmt.Errorf("no Location specified")
	}

	if s.DeviceType == "" {
		return fmt.Errorf("no DeviceType specified")
	}

	if s.DeviceUUID == "" {
		return fmt.Errorf("no DeviceUUID specified")
	}

	if len(s.ServiceList) == 0 {
		return fmt.Errorf("no ServiceList specified")
	}

	if s.Interface == nil {
		return fmt.Errorf("no interface specified")
	}

	if s.ServerHeader == "" {
		s.ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
	}

	if s.MaxAge == 0 {
		s.MaxAge = 30 * time.Minute
	}

	if s.NotifyInterval == 0 {
		s.NotifyInterval = 2 * s.MaxAge / 5
	}

	if s.MulticastTTL == 0 {
		s.MulticastTTL = 4
	}

	uuid := s.DeviceUUID
	if !strings.HasPrefix(uuid, "uuid:") {
		uuid = "uuid:" + uuid
	}
	s.targets = append([]string{uuid, "upnp:rootdevice", s.DeviceType}, s.ServiceList...)

	return nil
}

func (s *Server) notifyError(err error) error {
	if err != nil && s.ErrorHandler != nil {
		s.ErrorHandler(err, CallerName)
	}
	return err
}

func (s *Server) notifyInfo(msg string) {
	if s.InfoHandler != nil {
		s.InfoHandler(msg, CallerName)
	}
}
