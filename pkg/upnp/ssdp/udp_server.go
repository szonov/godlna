package ssdp

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
)

type UdpServer struct {
	// [page 12] To limit network congestion, the time-to-live (TTL) of each
	// IP packet for each multicast message should default to 4 and should be configurable.
	// Optional: Default is 4
	MulticastTTL int

	// Used network interface for SSDP server
	// I don't need multi-interface support for my purposes
	// Required: No defaults
	Interface *net.Interface

	// o SSDP options
	o *Options

	quit    chan struct{}
	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
}

func NewUdpServer(o *Options, iface *net.Interface, ttl ...int) *UdpServer {
	if len(ttl) == 0 {
		ttl = append(ttl, 4)
	}

	return &UdpServer{
		MulticastTTL: ttl[0],
		Interface:    iface,
		o:            o,
	}
}

func (s *UdpServer) validate() error {
	if s.MulticastTTL < 0 || s.MulticastTTL > 255 {
		s.MulticastTTL = 4
	}
	if s.Interface == nil {
		return fmt.Errorf("no network interface specified")
	}
	return s.o.Validate()
}

// Start the udp server, opens udp connection, listen for new messages and send responses
func (s *UdpServer) Start() error {
	var err error

	if err = s.validate(); err != nil {
		return err
	}

	if s.udpAddr, err = net.ResolveUDPAddr("udp4", MulticastAddrPort); err != nil {
		return err
	}

	if s.udpConn, err = net.ListenMulticastUDP("udp", s.Interface, s.udpAddr); err != nil {
		return err
	}

	if err = ipv4.NewPacketConn(s.udpConn).SetMulticastTTL(s.MulticastTTL); err != nil {
		return err
	}

	s.quit = make(chan struct{})
	go s.listen()

	runPeriodic(s.sendAlive, s.o.NotifyInterval, s.quit)

	return nil
}

// Stop sends bye message and close udp connection
func (s *UdpServer) Stop() error {
	close(s.quit)
	// run sendByeBye after closing quit channel,
	// make sure ByeBye message will be latest, and no one Alive message after it
	s.sendByeBye()
	return s.udpConn.Close()
}

// listen for incoming UDP messages
func (s *UdpServer) listen() {
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
			break
		}
		go s.parseUdpMessage(b[:n], addr)
	}
}

// parseUdpMessage processing message read from UDP connection
func (s *UdpServer) parseUdpMessage(buf []byte, sender *net.UDPAddr) {

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

	var targets []string
	if st == "ssdp:all" {
		targets = s.o.AllTargets()
	} else if s.o.HasTarget(st) {
		targets = []string{st}
	}

	if targets == nil || len(targets) == 0 {
		return
	}

	slog.Debug("ssdp:discover", slog.String("st", st), slog.String("sender", sender.String()))

	for _, target := range targets {
		msg := s.o.MSearchResponseMessage(target)
		delay := time.Duration(rand.Int63n(int64(time.Second) * mx))
		s.send(msg, sender, delay)
	}
}

// send write message to udp connection
func (s *UdpServer) send(msg []byte, to *net.UDPAddr, delay ...time.Duration) {
	if len(delay) > 0 {
		go func() {
			select {
			case <-s.quit:
			case <-time.After(delay[0]):
				s.send(msg, to)
			}
		}()
		return
	}
	if n, err := s.udpConn.WriteToUDP(msg, to); err != nil {
		slog.Warn("error writing to udp", slog.String("error", err.Error()))
	} else if n != len(msg) {
		slog.Warn("short write to udp", slog.Int("wrote", n), slog.Int("size", len(msg)))
	}
}

// sendAlive write Alive message to udp connection
func (s *UdpServer) sendAlive() {
	for _, target := range s.o.AllTargets() {
		// [page 15] Devices should wait a random interval less than 100 milliseconds before sending
		// an initial set of advertisements in order to reduce the likelihood of network storms;
		// this random interval should also be applied on occasions where the device obtains a
		// new IP address or a new network interface is installed.
		delay := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		s.send(s.o.AliveMessage(target), s.udpAddr, delay)
	}
}

// sendByeBye write Bye message to udp connection
func (s *UdpServer) sendByeBye() {
	for _, target := range s.o.AllTargets() {
		s.send(s.o.ByeByeMessage(target), s.udpAddr)
	}
}
