package upnp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	Identifier string = "UPnP"
)

type Controller interface {
	// OnServerStart initialize controller variables, which depends on server
	// Executed After Server created but before ListenAndServe
	OnServerStart(s *Server) error

	// Handle http request (if it handleable by controller)
	// returns:
	// - true - if request is handled by controller and next handlers should be skipped
	// - false - if request is not handleable by controller
	Handle(w http.ResponseWriter, r *http.Request) bool
}

type Server struct {
	ListenAddress string
	Controllers   []Controller

	// How to handle errors, useful for logs or something else.
	// Optional: No defaults
	ErrorHandler ErrorHandlerFunc

	// How to handle server notifications, useful for debug.
	// Optional: No defaults
	InfoHandler InfoHandlerFunc

	srv *http.Server
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

	s.srv = &http.Server{
		Addr:    s.ListenAddress,
		Handler: s,
	}

	for i := range s.Controllers {
		if err := s.Controllers[i].OnServerStart(s); err != nil {
			return s.notifyError(err)
		}
	}

	s.notifyInfo(fmt.Sprintf("starting UPnP server on address %s", s.ListenAddress))

	if err := s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return s.notifyError(err)
	}

	return nil
}

func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.notifyError(s.srv.Shutdown(ctx))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	s.notifyInfo(fmt.Sprintf("%s %s (%s)", r.Method, r.URL.Path, r.RemoteAddr))

	for i := range s.Controllers {
		if s.Controllers[i].Handle(w, r) {
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) validateAndSetDefaults() error {
	if s.ListenAddress == "" {
		return fmt.Errorf("no ListenAddress specified")
	}

	return nil
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
