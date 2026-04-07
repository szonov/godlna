package ssdp

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// reference: https://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.0-20080424.pdf

// Options common options for SSDP server
type Options struct {
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

	// targets all handled notification type(nt) / search target(st)
	// len = 3 of device + len(ServiceList)
	targets []string

	// cacheControl prepared string value for CACHE-CONTROL header
	cacheControl string
}

func (o *Options) UsnFromTarget(target string) string {
	if o.DeviceUDN == target {
		return target
	}
	return fmt.Sprintf("%s::%s", o.DeviceUDN, target)
}

func (o *Options) Validate() error {
	if o.Location == "" {
		return fmt.Errorf("no Location specified")
	}
	if o.DeviceType == "" {
		return fmt.Errorf("no DeviceType specified")
	}
	if o.DeviceUDN == "" {
		return fmt.Errorf("no Device UDN specified")
	}
	if o.ServerHeader == "" {
		o.ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
	}
	if o.MaxAge == 0 {
		o.MaxAge = 30 * time.Minute
	}
	o.cacheControl = "max-age=" + strconv.Itoa(int(o.MaxAge.Seconds()))
	if o.NotifyInterval == 0 {
		o.NotifyInterval = 2 * o.MaxAge / 5
	}
	if !strings.HasPrefix(o.DeviceUDN, "uuid:") {
		o.DeviceUDN = "uuid:" + o.DeviceUDN
	}
	o.targets = append([]string{o.DeviceUDN, rootDevice, o.DeviceType}, o.ServiceList...)

	return nil
}

func (o *Options) headers(statusLine string, headerPairs [][2]string) []byte {
	b := &bytes.Buffer{}

	write := func(w *bytes.Buffer, a ...any) {
		_, _ = fmt.Fprint(w, a...)
	}
	write(b, statusLine, "\r\n")
	for _, pair := range headerPairs {
		write(b, pair[0], ": ", pair[1], "\r\n")
	}
	write(b, "\r\n")
	return b.Bytes()
}

func (o *Options) AliveMessage(target string) []byte {
	return o.headers("NOTIFY * HTTP/1.1", [][2]string{
		{"HOST", MulticastAddrPort},
		{"CACHE-CONTROL", o.cacheControl},
		{"LOCATION", o.Location},
		{"SERVER", o.ServerHeader},
		{"NT", target},
		{"USN", o.UsnFromTarget(target)},
		{"NTS", Alive},
	})
}

func (o *Options) ByeByeMessage(target string) []byte {
	return o.headers("NOTIFY * HTTP/1.1", [][2]string{
		{"HOST", MulticastAddrPort},
		{"NT", target},
		{"USN", o.UsnFromTarget(target)},
		{"NTS", Bye},
	})
}

func (o *Options) MSearchResponseMessage(target string) []byte {
	return o.headers("HTTP/1.1 200 OK", [][2]string{
		{"CACHE-CONTROL", o.cacheControl},
		{"DATE", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")},
		{"ST", target},
		{"USN", o.UsnFromTarget(target)},
		{"EXT", ""},
		{"SERVER", o.ServerHeader},
		{"LOCATION", o.Location},
		{"Content-Length", "0"},
	})
}

func (o *Options) AllTargets() []string {
	return o.targets
}

func (o *Options) HasTarget(target string) bool {
	for _, t := range o.targets {
		if t == target {
			return true
		}
	}
	return false
}
