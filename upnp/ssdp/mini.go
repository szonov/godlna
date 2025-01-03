package ssdp

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// MiniServer Describes structure of SSDP Server which advertise (NOTIFY) by self
// and used minissdpd service for handling M-SEARCH requests
type MiniServer struct {
	Location     string
	ServerHeader string
	// MaxAge default is 30 minutes
	MaxAge time.Duration
	// NotifyInterval default is "2/5 * MaxAge"
	NotifyInterval time.Duration
	DeviceType     string
	DeviceUDN      string
	ServiceList    []string
	// MinissdpdSocket default is  "/var/run/minissdpd.sock"
	MinissdpdSocket string
	targets         []string
	quit            chan struct{}
	cacheControl    string
}

func (s *MiniServer) Start() error {
	var err error
	if err = s.validateAndSetDefaults(); err != nil {
		slog.Error(err.Error())
		return err
	}
	slog.Info("starting SSDP server")
	s.quit = make(chan struct{})
	if err = s.submitToMinissdpd(); err != nil {
		slog.Error(err.Error())
		return err
	}

	s.notifyAll(Alive)
	s.multicast()

	return nil
}

func (s *MiniServer) Stop() {
	slog.Info("stopping SSDP server")
	s.notifyAll(Bye)
	close(s.quit)
}

// multicast start notification daemon.
// Sends every Server.NotifyInterval new notifications about every targets
func (s *MiniServer) multicast() {
	tick := time.NewTicker(s.NotifyInterval)
	defer tick.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-tick.C:
		}
		s.notifyAll(Alive)
	}
}

func (s *MiniServer) validateAndSetDefaults() error {
	if s.Location == "" {
		return fmt.Errorf("no Location specified")
	}

	if s.DeviceType == "" {
		return fmt.Errorf("no DeviceType specified")
	}

	if s.DeviceUDN == "" {
		return fmt.Errorf("no DeviceUDN specified")
	}

	if s.ServerHeader == "" {
		s.ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
	}

	if s.MinissdpdSocket == "" {
		s.MinissdpdSocket = "/var/run/minissdpd.sock"
	}

	if s.MaxAge == 0 {
		s.MaxAge = 30 * time.Minute
	}
	s.cacheControl = "max-age=" + strconv.Itoa(int(s.MaxAge.Seconds()))
	if s.NotifyInterval == 0 {
		s.NotifyInterval = 2 * s.MaxAge / 5
	}

	if !strings.HasPrefix(s.DeviceUDN, "uuid:") {
		s.DeviceUDN = "uuid:" + s.DeviceUDN
	}

	s.targets = append([]string{s.DeviceUDN, "upnp:rootdevice", s.DeviceType}, s.ServiceList...)

	return nil
}

func (s *MiniServer) notifyAll(nts string) {
	slog.Debug("notifying ssdp", "nts", nts)
	conn, err := net.Dial("udp", MulticastAddrPort)
	if err != nil {
		err = fmt.Errorf("can not connect to multicast address %s, %w", MulticastAddrPort, err)
		slog.Error(err.Error())
		return
	}
	defer func(conn net.Conn) {
		if err = conn.Close(); err != nil {
			err = fmt.Errorf("can not close connect to multicast address %s, %w", MulticastAddrPort, err)
			slog.Error(err.Error())
		}
	}(conn)

	for _, target := range s.targets {
		//delay := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		msg := s.makeMessage(target, nts)
		if _, err = conn.Write([]byte(msg)); err != nil {
			err = fmt.Errorf("can not send message to multicast address %s, %w", MulticastAddrPort, err)
			slog.Error(err.Error())
		}
	}
}

func (s *MiniServer) usnFromTarget(target string) string {
	if s.DeviceUDN == target {
		return s.DeviceUDN
	}
	return s.DeviceUDN + "::" + target
}

func (s *MiniServer) makeMessage(target string, nts string) string {
	return "NOTIFY * HTTP/1.1\r\n" +
		"HOST: " + MulticastAddrPort + "\r\n" +
		"CACHE-CONTROL: " + s.cacheControl + "\r\n" +
		"LOCATION: " + s.Location + "\r\n" +
		"SERVER: " + s.ServerHeader + "\r\n" +
		"NT: " + target + "\r\n" +
		"USN: " + s.usnFromTarget(target) + "\r\n" +
		"NTS: " + nts + "\r\n" +
		"\r\n"
}

func (s *MiniServer) submitToMinissdpd() error {
	slog.Debug("submitting to minissdpd socket", "socket", s.MinissdpdSocket)
	var err error

	var minissdpd net.Conn
	if minissdpd, err = net.Dial("unix", s.MinissdpdSocket); err != nil {
		return err
	}
	defer func(minissdpd net.Conn) {
		err = minissdpd.Close()
		if err != nil {
			slog.Error("error closing minissdpd", slog.String("error", err.Error()))
		}
	}(minissdpd)

	for _, target := range s.targets {
		buf := &bytes.Buffer{}
		usn := s.usnFromTarget(target)

		slog.Info("MINISSDPD", "target", target)

		_, err = fmt.Fprintf(buf, "\x04%c%s%c%s%c%s%c%s",
			len(target), target,
			len(usn), usn,
			len(s.ServerHeader), s.ServerHeader,
			len(s.Location), s.Location,
		)
		if err != nil {
			return err
		}
		if _, err = minissdpd.Write(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}
