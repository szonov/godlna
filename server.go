package upnp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/szonov/go-upnp-lib/soap"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/szonov/go-upnp-lib/device"
	"github.com/szonov/go-upnp-lib/ssdp"
)

type Controller interface {
	// OnServerStart initialize controller variables, which depends on server
	// Executed After Device created, but before ListenAndServe
	// Good place to add services to Device and setup routes
	OnServerStart(s *Server) error
}

type Server struct {
	// Format ip:port
	ListenAddress string

	// Direct creation is not allowed, use OnDeviceCreate callback
	DeviceDescription *device.Description

	// OnDeviceCreate runs after device created and assigned to server
	// Good place to add own fields to Device
	OnDeviceCreate func(*device.Description) error

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

	BeforeHook func(w http.ResponseWriter, r *http.Request) bool // true = continue execution, false = stop
	AfterHook  func(w http.ResponseWriter, r *http.Request)

	// private
	srv           *http.Server
	router        *http.ServeMux
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
		slog.Error(err.Error())
		return err
	}

	// create device, and modify it using callback OnDeviceCreate
	if err = s.makeDeviceDescription(); err != nil {
		slog.Error(err.Error())
		return err
	}

	s.router = http.NewServeMux()
	s.srv = &http.Server{
		Addr:    s.ListenAddress,
		Handler: s.router,
	}

	if err = s.setupRoutes(); err != nil {
		slog.Error(err.Error())
		return err
	}

	// prepare device desc xml
	if err = s.makeDeviceDescXML(); err != nil {
		slog.Error(err.Error())
		return err
	}

	s.startSsdpServer()

	slog.Info("starting UPnP server", slog.String("address", s.ListenAddress))

	if err = s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
		return err
	}

	return nil
}

func (s *Server) Shutdown() {
	if s.ssdpServer != nil {
		s.ssdpServer.Shutdown()
	}

	slog.Info("stopping UPnP server", slog.String("address", s.ListenAddress))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error(err.Error())
	}
}

func (s *Server) Handle(pattern string, handlerFunc http.HandlerFunc) {
	s.router.Handle(pattern, s.hook(handlerFunc))
}

func (s *Server) AppendService(service *device.Service) {
	s.DeviceDescription.Device.ServiceList = append(s.DeviceDescription.Device.ServiceList, service)
}

func (s *Server) setupRoutes() error {

	for i := range s.Controllers {
		if err := s.Controllers[i].OnServerStart(s); err != nil {
			slog.Error(err.Error())
			return err
		}
	}

	// root desc
	s.Handle(s.DeviceDescription.Location, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			w.Header().Set("Content-Type", soap.ResponseContentTypeXML)
			_, _ = w.Write(s.deviceDescXML)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return nil
}

func (s *Server) hook(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.BeforeHook != nil {
			if !s.BeforeHook(w, r) {
				return
			}
		}
		w.Header().Set("Server", s.ServerHeader)
		next.ServeHTTP(w, r)
		if s.AfterHook != nil {
			s.AfterHook(w, r)
		}
	})
}

func (s *Server) validateAndSetDefaults() error {
	if s.ListenAddress == "" {
		return fmt.Errorf("no ListenAddress specified")
	}
	if s.ServerHeader == "" {
		s.ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
	}
	return nil
}

func (s *Server) makeDeviceDescription() error {

	friendlyName := DefaultFriendlyName()

	s.DeviceDescription = &device.Description{
		SpecVersion: device.SpecVersion{Major: 1},
		Device: &device.Device{
			DeviceType:   DefaultDeviceType,
			FriendlyName: friendlyName,
			UDN:          NewUDN(friendlyName),
			Manufacturer: DefaultManufacturer,
			ModelName:    DefaultModelName,
			ServiceList:  make([]*device.Service, 0),
		},
		Location: "/rootDesc.xml",
	}

	// Possibility to set correct properties to device
	if s.OnDeviceCreate != nil {
		if err := s.OnDeviceCreate(s.DeviceDescription); err != nil {
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
	for _, serv := range s.DeviceDescription.Device.ServiceList {
		services = append(services, serv.ServiceType)
	}

	s.ssdpServer = &ssdp.Server{
		Location:       "http://" + s.ListenAddress + s.DeviceDescription.Location,
		ServerHeader:   s.ServerHeader,
		DeviceType:     s.DeviceDescription.Device.DeviceType,
		DeviceUDN:      s.DeviceDescription.Device.UDN,
		ServiceList:    services,
		Interface:      s.SsdpInterface,
		MaxAge:         s.SsdpMaxAge,
		NotifyInterval: s.SsdpNotifyInterval,
		MulticastTTL:   s.SsdpMulticastTTL,
	}

	s.ssdpServer.Start()
}

func (s *Server) makeDeviceDescXML() (err error) {
	var b []byte
	if b, err = xml.Marshal(s.DeviceDescription); err == nil {
		s.deviceDescXML = append([]byte(xml.Header), b...)
	}
	return
}
