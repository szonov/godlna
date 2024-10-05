package main

import (
	"github.com/szonov/go-upnp-lib/device"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/presentation"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/contentdirectory"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/logger"
	"github.com/szonov/go-upnp-lib/network"
)

func main() {

	logger.InitLogger()

	v4face := network.DefaultV4Interface()

	upnpServer := &upnp.Server{
		ListenAddress: v4face.IP + ":55975",
		SsdpInterface: v4face.Interface,
		Controllers: []upnp.Controller{
			contentdirectory.NewServiceController(),
			presentation.NewController(),
		},
		OnDeviceCreate: func(d *device.Description) error {
			slog.Debug("call:OnDeviceCreate (time to setup Device)", slog.String("name", d.Device.FriendlyName))
			return nil
		},
		BeforeHook: func(w http.ResponseWriter, r *http.Request) bool {
			slog.Debug("Request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote", r.RemoteAddr),
			)
			logger.DebugRequest(r)
			return true
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
