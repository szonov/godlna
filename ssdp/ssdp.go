package ssdp

import (
	"time"

	"github.com/szonov/go-upnp-lib"
)

const (
	MulticastAddrPort string = "239.255.255.250:1900"
	CallerName        string = "ssdp"
)

type Server struct {
	ErrorHandler   upnp.ErrorHandlerFunc
	InfoHandler    upnp.InfoHandlerFunc
	NotifyInterval time.Duration
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
