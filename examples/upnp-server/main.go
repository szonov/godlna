package main

import (
	"github.com/szonov/go-upnp-lib/examples/upnp-server/contentdirectory"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/presentation"
	"github.com/szonov/go-upnp-lib/network"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/logger"
)

func main() {

	logger.InitLogger()

	v4face := network.DefaultV4Interface()
	listenAddress := v4face.ListenAddress(55975)
	upnpServer := &upnp.Server{
		ListenAddress: listenAddress,
		SsdpInterface: v4face.Interface,
		Controllers: []upnp.Controller{
			contentdirectory.NewServiceController(),
			new(presentation.Controller),
		},
		//DeviceDescription: upnp.DefaultDeviceDesc().With(func(desc *upnp.DeviceDescription) {
		//	desc.Device.PresentationURL = "http://" + listenAddress + "/"
		//}),
		Middleware: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				slog.Debug("Request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote", r.RemoteAddr),
				)
				logger.DebugRequest(r)
				next.ServeHTTP(w, r)
			})
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
