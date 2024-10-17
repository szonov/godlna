package dlnaserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/szonov/godlna/upnp/device"
	"github.com/szonov/godlna/upnp/ssdp"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"time"
)

type (
	Server struct {
		// ListenAddress Format ip:port
		ListenAddress string

		// DeviceDescription description of device, required only if defined SsdpInterface
		DeviceDescription *device.Description

		// Optional: Default is "[runtime.GOOS]/[runtime.Version()] UPnP/1.0 GoUPnP/1.0"
		ServerHeader string

		// SsdpInterface Interface, on which start SSDP server
		// If not defined ssdp server will be disabled
		SsdpInterface *net.Interface

		// BeforeStart runs before http and ssdp server starts,
		// place to set up routes, modify configuration of servers
		BeforeStart func(*Server, *http.ServeMux, *ssdp.Server)

		// OnHttpRequest possibility to inject own code to request processor
		OnHttpRequest func(*Server, http.HandlerFunc, http.ResponseWriter, *http.Request)

		// private
		srv        *http.Server
		ssdpServer *ssdp.Server
	}
)

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
		err = fmt.Errorf("no ListenAddress specified")
		slog.Error(err.Error())
		return err
	}

	if s.ServerHeader == "" {
		s.ServerHeader = DefaultServerHeader()
	}

	if s.DeviceDescription == nil && s.SsdpInterface != nil {
		err = fmt.Errorf("DeviceDescription required for creating SSDP server")
		slog.Error(err.Error())
		return err
	}

	if s.SsdpInterface != nil {
		s.makeSsdpServer()
	}

	mux := http.NewServeMux()
	s.srv = &http.Server{
		Addr:    s.ListenAddress,
		Handler: mux,
	}

	if s.BeforeStart != nil {
		s.BeforeStart(s, mux, s.ssdpServer)
	}

	if s.ssdpServer != nil {
		s.ssdpServer.Start()
	}

	slog.Info("starting HTTP server", slog.String("address", s.ListenAddress))

	if err = s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
		return err
	}

	return nil
}

// // HookHandler middleware
//
//	func (s *Server) HookHandler(next http.HandlerFunc) http.Handler {
//		return s.Hook(next)
//	}
//

// Hook middleware adds 'Server' header to outgoing responses
func (s *Server) Hook(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.OnHttpRequest != nil {
			s.OnHttpRequest(s, next, w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (s *Server) Shutdown() {
	if s.ssdpServer != nil {
		s.ssdpServer.Shutdown()
	}

	slog.Info("stopping HTTP server", slog.String("address", s.ListenAddress))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error(err.Error())
	}
}

func (s *Server) makeSsdpServer() {

	services := make([]string, 0)
	for _, serv := range s.DeviceDescription.Device.ServiceList {
		services = append(services, serv.ServiceType)
	}

	s.ssdpServer = &ssdp.Server{
		Location:     "http://" + s.ListenAddress + s.DeviceDescription.Location,
		ServerHeader: s.ServerHeader,
		DeviceType:   s.DeviceDescription.Device.DeviceType,
		DeviceUDN:    s.DeviceDescription.Device.UDN,
		ServiceList:  services,
		Interface:    s.SsdpInterface,
	}
}

func DefaultServerHeader() string {
	return fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")
}
