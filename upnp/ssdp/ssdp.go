package ssdp

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"log/slog"
	"math/rand"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	MulticastAddrPort string = "239.255.255.250:1900"
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

	// [page 12] To limit net_utils congestion, the time-to-live (TTL) of each
	// IP packet for each multicast message should default to 4 and should be configurable.
	// Optional: Default is 4
	MulticastTTL int

	// Full device type. Should contain exact value, added to xml in <deviceType>.*</deviceType>
	// Example: "urn:schemas-upnp-org:device:MediaServer:1"
	// Required: No defaults
	DeviceType string

	// Device UDN specified by UPnP vendor. (with or without "uuid:" prefix)
	// Valid examples:
	// - "uuid:da2cc462-0000-0000-0000-44fd2452e03f"
	// - "da2cc462-0000-0000-0000-44fd2452e03f"
	// Required: No defaults
	DeviceUDN string

	// List of full service types as it appears in xml in <serviceType>.*</serviceType>
	// Example: []string{
	//   "urn:schemas-upnp-org:service:ContentDirectory:1",
	//   "urn:schemas-upnp-org:service:ConnectionManager:1",
	// }
	// Optional: No defaults
	ServiceList []string

	// Used net_utils interface for ssdp server
	// I don't need multi-interface support for my purposes
	// Required: No defaults
	Interface *net.Interface

	// all handled notification type(nt) / search target(st)
	// len = 3 of device + len(ServiceList)
	targets []string

	quit         chan struct{}
	udpAddr      *net.UDPAddr
	udpConn      *net.UDPConn
	cacheControl string
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
		slog.Error(err.Error())
		return err
	}

	slog.Info("starting SSDP server",
		slog.String("address", MulticastAddrPort),
		slog.String("if", s.Interface.Name),
	)

	if s.udpAddr, err = net.ResolveUDPAddr("udp4", MulticastAddrPort); err != nil {
		slog.Error(err.Error())
		return err
	}

	s.udpConn, err = net.ListenMulticastUDP("udp", s.Interface, s.udpAddr)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	if err = ipv4.NewPacketConn(s.udpConn).SetMulticastTTL(s.MulticastTTL); err != nil {
		slog.Error(err.Error())
		return err
	}

	s.quit = make(chan struct{})
	go s.listen()
	s.sendAlive()
	s.multicast()

	return nil
}

func (s *Server) Shutdown() {
	slog.Info("stopping SSDP server",
		slog.String("address", MulticastAddrPort),
		slog.String("if", s.Interface.Name),
	)
	close(s.quit)
	s.sendByeBye()
	if err := s.udpConn.Close(); err != nil {
		slog.Error(err.Error())
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
		if err != nil {
			slog.Error(err.Error())
			break
		}
		go s.parseUdpMessage(b[:n], addr)
	}
}

// multicast start notification daemon.
// Sends every Server.NotifyInterval new notifications about every targets
func (s *Server) multicast() {

	tick := time.NewTicker(s.NotifyInterval)
	defer tick.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-tick.C:
		}
		s.sendAlive()
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
				if err != nil {
					slog.Warn("invalid MX", slog.String("err", err.Error()), slog.String("line", line))
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

	slog.Debug("ssdp:discover",
		slog.String("st", st),
		slog.String("sender", sender.String()),
	)

	for _, target := range targets {
		msg := s.makeMSearchResponse(target)
		delay := time.Duration(rand.Int63n(int64(time.Second) * mx))
		s.send(msg, sender, delay)
	}
}

func (s *Server) send(msg string, to *net.UDPAddr, delay ...time.Duration) {
	if len(delay) > 0 {
		go func() {
			select {
			case <-time.After(delay[0]):
				s.send(msg, to)
			case <-s.quit:
			}
		}()
		return
	}
	buf := []byte(msg)
	if n, err := s.udpConn.WriteToUDP(buf, to); err != nil {
		slog.Debug("error writing to udp", slog.String("error", err.Error()))
	} else if n != len(buf) {
		slog.Debug("short write to udp", slog.Int("wrote", n), slog.Int("size", len(buf)))
	}
}

func (s *Server) sendAlive() {
	for _, target := range s.targets {
		msg := s.makeAliveMessage(target)
		// [page 15] Devices should wait a random interval less than 100 milliseconds before sending
		// an initial set of advertisements in order to reduce the likelihood of net_utils storms;
		// this random interval should also be applied on occasions where the device obtains a
		// new IP address or a new net_utils interface is installed.
		delay := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		s.send(msg, s.udpAddr, delay)
	}
}

func (s *Server) sendByeBye() {
	for _, target := range s.targets {
		msg := s.makeByeByeMessage(target)
		s.send(msg, s.udpAddr)
	}
}

func (s *Server) usnFromTarget(target string) string {
	if s.DeviceUDN == target {
		return s.DeviceUDN
	}
	return s.DeviceUDN + "::" + target
}

func (s *Server) makeAliveMessage(target string) string {
	return "NOTIFY * HTTP/1.1\r\n" +
		"HOST: " + MulticastAddrPort + "\r\n" +
		"CACHE-CONTROL: " + s.cacheControl + "\r\n" +
		"LOCATION: " + s.Location + "\r\n" +
		"SERVER: " + s.ServerHeader + "\r\n" +
		"NT: " + target + "\r\n" +
		"USN: " + s.usnFromTarget(target) + "\r\n" +
		"NTS: ssdp:alive\r\n"
}

func (s *Server) makeByeByeMessage(target string) string {
	return "NOTIFY * HTTP/1.1\r\n" +
		"HOST: " + MulticastAddrPort + "\r\n" +
		"NT: " + target + "\r\n" +
		"USN: " + s.usnFromTarget(target) + "\r\n" +
		"NTS: ssdp:byebye\r\n"
}

func (s *Server) makeMSearchResponse(target string) string {
	return "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: " + s.cacheControl + "\r\n" +
		"DATE: " + time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT") + "\r\n" +
		"ST: " + target + "\r\n" +
		"USN: " + s.usnFromTarget(target) + "\r\n" +
		"EXT:\r\n" +
		"SERVER: " + s.ServerHeader + "\r\n" +
		"LOCATION: " + s.Location + "\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"
}

func (s *Server) validateAndSetDefaults() error {
	if s.Location == "" {
		return fmt.Errorf("no Location specified")
	}

	if s.DeviceType == "" {
		return fmt.Errorf("no DeviceType specified")
	}

	if s.DeviceUDN == "" {
		return fmt.Errorf("no DeviceUDN specified")
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

	s.cacheControl = "max-age=" + strconv.Itoa(int(s.MaxAge.Seconds()))

	if s.NotifyInterval == 0 {
		s.NotifyInterval = 2 * s.MaxAge / 5
	}

	if s.MulticastTTL == 0 {
		s.MulticastTTL = 4
	}

	if !strings.HasPrefix(s.DeviceUDN, "uuid:") {
		s.DeviceUDN = "uuid:" + s.DeviceUDN
	}
	s.targets = append([]string{s.DeviceUDN, "upnp:rootdevice", s.DeviceType}, s.ServiceList...)

	return nil
}
