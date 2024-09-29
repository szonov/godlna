package upnp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/szonov/go-upnp-lib/ssdp"
	"net"
	"net/http"
	"runtime"
	"time"
)

const (
	Identifier             string = "UPnP"
	ResponseContentTypeXML        = `text/xml; charset="utf-8"`
)

type Controller interface {
	// OnServerStart initialize controller variables, which depends on server
	// Executed After Device created, but before ListenAndServe
	// Good place to add services to Device
	OnServerStart(s *Server) error

	// Handle http request (if it handleable by controller)
	// returns:
	// - true - if request is handled by controller and next handlers should be skipped
	// - false - if request is not handleable by controller
	Handle(w http.ResponseWriter, r *http.Request) bool
}

type Server struct {
	// Format ip:port
	ListenAddress string

	// Direct creation is not allowed, use OnDeviceCreate callback
	Device *Device

	// Optional: Default is "/rootDesc.xml"
	DeviceDescPath string

	// OnDeviceCreate runs after device created and assigned to server
	// Good place to add own fields to Device
	OnDeviceCreate func(*Server) error

	Controllers []Controller

	// Optional: Default is "[runtime.GOOS]/[runtime.Version()] UPnP/1.0 GoUPnP/1.0"
	ServerHeader string

	// SsdpInterface Interface, on which start SSDP server
	// If not provided, build-in SSDP server will be disabled, and you can start another ssdp server
	// Optional: No defaults
	SsdpInterface *net.Interface

	// SsdpMaxAge MaxAge parameter for build-in ssdp server, without SsdpInterface it is useful
	SsdpMaxAge time.Duration

	// SsdpNotifyInterval NotifyInterval parameter for build-in ssdp server, without SsdpInterface it is useful
	SsdpNotifyInterval time.Duration

	// SsdpMulticastTTL MulticastTTL parameter for build-in ssdp server, without SsdpInterface it is useful
	SsdpMulticastTTL int

	// How to handle errors, useful for logs or something else.
	// Optional: No defaults
	ErrorHandler ErrorHandlerFunc

	// How to handle server notifications, useful for debug.
	// Optional: No defaults
	InfoHandler InfoHandlerFunc

	srv           *http.Server
	deviceDescXML []byte
	ssdpServer    *ssdp.Server
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

	// create device, and modify it using callback OnDeviceCreate
	if err = s.makeDevice(); err != nil {
		return s.notifyError(err)
	}

	// initialize all controllers
	for i := range s.Controllers {
		if err = s.Controllers[i].OnServerStart(s); err != nil {
			return s.notifyError(err)
		}
	}

	// prepare device desc xml
	if err = s.makeDeviceDescXML(); err != nil {
		return err
	}

	s.startSsdpServer()

	s.notifyInfo(fmt.Sprintf("starting UPnP server on address %s", s.ListenAddress))

	s.srv = &http.Server{
		Addr:    s.ListenAddress,
		Handler: s,
	}

	if err = s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return s.notifyError(err)
	}

	return nil
}

func (s *Server) Shutdown() {
	if s.ssdpServer != nil {
		s.ssdpServer.Shutdown()
	}
	s.notifyInfo(fmt.Sprintf("stopping UPnP server on address %s", s.ListenAddress))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.notifyError(s.srv.Shutdown(ctx))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	s.notifyInfo(fmt.Sprintf("%s %s (%s)", r.Method, r.URL.Path, r.RemoteAddr))

	w.Header().Set("Server", s.ServerHeader)

	for i := range s.Controllers {
		if s.Controllers[i].Handle(w, r) {
			return
		}
	}

	if r.URL.Path == s.DeviceDescPath {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			w.Header().Set("Content-Type", ResponseContentTypeXML)
			_, _ = w.Write(s.deviceDescXML)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) validateAndSetDefaults() error {
	if s.ListenAddress == "" {
		return fmt.Errorf("no ListenAddress specified")
	}
	if s.DeviceDescPath == "" {
		s.DeviceDescPath = "/rootDesc.xml"
	}
	if s.ServerHeader == "" {
		s.ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
	}
	return nil
}

func (s *Server) makeDevice() error {

	friendlyName := DefaultFriendlyName()

	s.Device = &Device{
		DeviceType:   DefaultDeviceType,
		FriendlyName: friendlyName,
		UDN:          NewUDN(friendlyName),
		Manufacturer: DefaultManufacturer,
		ModelName:    DefaultModelName,
	}

	// Possibility to set correct properties to device
	if s.OnDeviceCreate != nil {
		if err := s.OnDeviceCreate(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) startSsdpServer() {
	if s.SsdpInterface == nil {
		return
	}

	services := make([]string, 0)
	for _, serv := range s.Device.ServiceList {
		services = append(services, serv.ServiceType)
	}

	s.ssdpServer = &ssdp.Server{
		Location:       "http://" + s.ListenAddress + s.DeviceDescPath,
		ServerHeader:   s.ServerHeader,
		DeviceType:     s.Device.DeviceType,
		DeviceUDN:      s.Device.UDN,
		ServiceList:    services,
		Interface:      s.SsdpInterface,
		MaxAge:         s.SsdpMaxAge,
		NotifyInterval: s.SsdpNotifyInterval,
		MulticastTTL:   s.SsdpMulticastTTL,
		ErrorHandler:   s.ErrorHandler,
		InfoHandler:    s.InfoHandler,
	}

	s.ssdpServer.Start()
}

func (s *Server) makeDeviceDescXML() (err error) {
	var b []byte
	deviceDesc := DeviceDesc{
		SpecVersion: SpecVersion{Major: 1},
		Device:      *s.Device,
	}
	if b, err = xml.Marshal(deviceDesc); err == nil {
		s.deviceDescXML = append([]byte(xml.Header), b...)
	}
	return
}

func (s *Server) notifyError(err error) error {
	if err != nil && s.ErrorHandler != nil {
		s.ErrorHandler(err, Identifier)
	}
	return err
}

func (s *Server) notifyInfo(msg string) {
	if s.InfoHandler != nil {
		s.InfoHandler(msg, Identifier)
	}
}
