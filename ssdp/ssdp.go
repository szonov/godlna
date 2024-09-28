package ssdp

import (
	"net"
	"time"

	"github.com/szonov/go-upnp-lib"
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
	// Forexample, control points implementing UDA version 1.0 will be able to interoperate with devices
	// implementing UDA version 1.1.
	// Example: "Linux/6.0 UPnP/1.0 App/1.0"
	// Optional: Default is "TODO..."
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
	// Optional: Default is "TODO..."
	NotifyInterval time.Duration

	// Device UUID specified by UPnP vendor. (with or without "uuid:" prefix)
	// Valid examples:
	// - "uuid:da2cc462-0000-0000-0000-44fd2452e03f"
	// - "da2cc462-0000-0000-0000-44fd2452e03f"
	// Required: No defaults
	DeviceUUID string

	// Full device type. Should contain exact value, added to xml in <deviceType>.*</deviceType>
	// Example: "urn:schemas-upnp-org:device:MediaServer:1"
	// Required: No defaults
	DeviceType string

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
	Interface net.Interface

	// How to handle errors, useful for logs or something else.
	// Optional: No defaults
	ErrorHandler upnp.ErrorHandlerFunc

	// How to handle ssdp server notifications, useful for debug.
	// Optional: No defaults
	InfoHandler upnp.InfoHandlerFunc
}

func (s *Server) Start() *Server {
	go func() {
		if err := s.ListenAndServe(); err != nil {
			s.notifyError(err)
			panic(err)
		}
	}()
	return s
}

func (s *Server) ListenAndServe() error {
	return nil
}

func (s *Server) Shutdown() {
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
