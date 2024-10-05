package main

import (
	"log/slog"
	"os"
	"os/signal"

	"github.com/szonov/go-upnp-lib/examples/upnp-server/contentdirectory"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/logger"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/presentation"

	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/network"
)

func main() {

	logger.InitLogger()

	v4face := network.DefaultV4Interface()

	upnpServer := &upnp.Server{
		ListenAddress: v4face.IP + ":55975",
		SsdpInterface: v4face.Interface,
		Controllers: []upnp.Controller{
			logger.NewDebugController(),
			presentation.NewController(),
			contentdirectory.NewServiceController(),
		},
		OnDeviceCreate: func(s *upnp.Server) error {
			slog.Debug("call:OnDeviceCreate (time to setup Device)")
			return nil
		},
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		// terminate ssdp server
		slog.Debug("gracefully shutting down...")
		upnpServer.Shutdown()
	}()

	slog.Info("server starting...")

	_ = upnpServer.ListenAndServe()

	slog.Info("server stopped")
}
