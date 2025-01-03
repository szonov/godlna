package dlna

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/szonov/godlna/logger"
	"github.com/szonov/godlna/upnp/device"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var ServerHeader = fmt.Sprintf("%s/%s %s %s", runtime.GOOS, runtime.Version(), "UPnP/1.0", "GoUPnP/1.0")

type Server struct {
	ListenAddress      string
	DeviceDescription  *device.Description
	RootDirectory      string
	DebugRequest       bool
	DebugRequestHeader bool
	DebugRequestBody   bool
	Psql               *pgxpool.Pool
	srv                *http.Server
}

func NewServer(root string, friendlyName string, listenAddr string, psql *pgxpool.Pool) *Server {
	return &Server{
		ListenAddress:     listenAddr,
		RootDirectory:     root,
		DeviceDescription: makeDeviceDescription(friendlyName, listenAddr),
		Psql:              psql,
	}
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
		err = fmt.Errorf("no ListenAddress specified")
		slog.Error(err.Error())
		return err
	}

	if s.DeviceDescription == nil {
		err = fmt.Errorf("no DeviceDescription specified")
		slog.Error(err.Error())
		return err
	}

	mux := http.NewServeMux()
	s.srv = &http.Server{
		Addr:    s.ListenAddress,
		Handler: mux,
	}

	if err = s.setupRoutes(mux); err != nil {
		slog.Error(err.Error())
		return err
	}

	slog.Info("starting HTTP server", slog.String("address", s.ListenAddress))

	if err = s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
		return err
	}

	return nil
}

func (s *Server) Shutdown() {
	slog.Info("stopping HTTP server", slog.String("address", s.ListenAddress))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error(err.Error())
	}
}

func (s *Server) hook(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.DebugRequest {
			logger.DebugRequest(r, s.DebugRequestHeader, s.DebugRequestBody)
		}
		w.Header().Set("Server", ServerHeader)
		next.ServeHTTP(w, r)
	}
}

func (s *Server) setupRoutes(mux *http.ServeMux) error {
	var err error
	var deviceController *DeviceController
	var cdsController *ContentDirectoryController

	if deviceController, err = NewDeviceController(s); err != nil {
		return err
	}

	if cdsController, err = NewContentDirectoryController(s); err != nil {
		return err
	}

	// index
	mux.HandleFunc("/", s.hook(deviceController.HandleIndexURL))

	// device
	mux.HandleFunc("/device/desc.xml", s.hook(deviceController.HandleDescriptionURL))
	mux.HandleFunc("/device/icons/", s.hook(deviceController.HandleIcons))

	// content directory
	mux.HandleFunc("/cds/desc.xml", s.hook(cdsController.HandleSCPDURL))
	mux.HandleFunc("/cds/ctl", s.hook(cdsController.HandleControlURL))
	mux.HandleFunc("/cds/evt", s.hook(cdsController.HandleEventSubURL))

	// content
	mux.HandleFunc("/ct/t/{obj}", s.hook(cdsController.HandleContentURL))
	mux.HandleFunc("/ct/v/{obj}", s.hook(cdsController.HandleContentURL))

	return nil
}

func useSecondsInBookmark(r *http.Request) bool {
	agent := r.Header.Get("User-Agent")
	return agent == "DLNADOC/1.50" || strings.Contains(agent, "40C7000")
}
