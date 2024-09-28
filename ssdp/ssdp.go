package ssdp

import (
	"fmt"
	"net"
	"runtime"
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
	if err := s.validateAndSetDefaults(); err != nil {
		return s.notifyError(err)
	}

	msg := fmt.Sprintf("starting ssdp server on address %s (%s)", MulticastAddrPort, s.Interface.Name)
	s.notifyInfo(msg)

	return nil
}

func (s *Server) Shutdown() {
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

	uuid := s.DeviceUUID
	if !strings.HasPrefix(uuid, "uuid:") {
		uuid = "uuid:" + uuid
	}
	s.targets = append([]string{uuid, "upnp:rootdevice", s.DeviceUUID}, s.ServiceList...)

	return nil
}

func (s *Server) notifyError(err error) error {
	if s.ErrorHandler != nil {
		s.ErrorHandler(err, CallerName)
	}
	return err
}

func (s *Server) notifyInfo(msg string) {
	if s.InfoHandler != nil {
		s.InfoHandler(msg, CallerName)
	}
}
