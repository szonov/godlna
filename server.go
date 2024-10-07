package upnp

import (
	"context"
	"errors"
	"fmt"
	"github.com/szonov/go-upnp-lib/network"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/szonov/go-upnp-lib/ssdp"
)

type Route struct {
	Pattern    string
	HandleFunc http.HandlerFunc
}

type Controller interface {
	// RegisterRoutes initialize controller variables, which depends on device
	// and register handled routes. Executed on server start,
	// Good place to add services to Device, modify device settings
	RegisterRoutes(deviceDesc *DeviceDescription) ([]Route, error)
}

type Server struct {
	// ListenAddress Format ip:port, by default selected automatically
	ListenAddress string

	// Description of device, by default generated automatically
	DeviceDescription *DeviceDescription

	// All controllers (DeviceController added automatically to this list on starting server)
	Controllers []Controller

	// Optional: Default is "[runtime.GOOS]/[runtime.Version()] UPnP/1.0 GoUPnP/1.0"
	ServerHeader string

	// SsdpInterface Interface, on which start SSDP server
	// If ListenAddress not defined, then ListenAddress and SsdpInterface chosen automatically
	// To disable ssdp server you should define ListenAddress and skip setup of SsdpInterface
	SsdpInterface *net.Interface

	// SsdpMaxAge MaxAge parameter for build-in ssdp server, without SsdpInterface it is useful
	SsdpMaxAge time.Duration

	// SsdpNotifyInterval NotifyInterval parameter for build-in ssdp server, without SsdpInterface it is useful
	SsdpNotifyInterval time.Duration

	// SsdpMulticastTTL MulticastTTL parameter for build-in ssdp server, without SsdpInterface it is useful
	SsdpMulticastTTL int

	// Middleware own handler... do not forget to include next.ServeHTTP(w, r) inside
	Middleware func(http.Handler) http.Handler

	// private
	srv        *http.Server
	ssdpServer *ssdp.Server
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

	if s.ListenAddress == "" {
		// try to detect automatically
		if s.SsdpInterface != nil {
			s.ListenAddress = network.DefaultV4Interface(s.SsdpInterface.Name).ListenAddress()
		} else {
			v4face := network.DefaultV4Interface()
			s.ListenAddress = v4face.ListenAddress()
			s.SsdpInterface = v4face.Interface
		}
		if s.ListenAddress == "" {
			err = fmt.Errorf("network configuration problem")
			slog.Error(err.Error())
			return err
		}
		slog.Info("network chosen automatically", "addr", s.ListenAddress)
	}

	if s.ServerHeader == "" {
		s.ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
	}

	// make default DeviceDescription if not defined
	if s.DeviceDescription == nil {
		s.DeviceDescription = DefaultDeviceDesc().With(func(desc *DeviceDescription) {
			desc.Device.PresentationURL = "http://" + s.ListenAddress + "/"
		})
	}

	// add controller with device description to the last position,
	// make sure all controllers made manipulations with device and it has actual status
	s.Controllers = append(s.Controllers, new(DeviceController))

	mux := http.NewServeMux()
	for i := range s.Controllers {
		var routes []Route
		if routes, err = s.Controllers[i].RegisterRoutes(s.DeviceDescription); err != nil {
			slog.Error(err.Error())
			return err
		}
		for _, route := range routes {
			handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Server", s.ServerHeader)
				route.HandleFunc.ServeHTTP(w, r)
			}))
			if s.Middleware != nil {
				handler = s.Middleware(handler)
			}
			mux.Handle(route.Pattern, handler)
		}
	}

	s.startSsdpServer()

	s.srv = &http.Server{
		Addr:    s.ListenAddress,
		Handler: mux,
	}

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
