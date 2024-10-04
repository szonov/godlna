package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/szonov/go-upnp-lib/examples/upnp-server/contentdirectory"
	"github.com/szonov/go-upnp-lib/examples/upnp-server/presentation"

	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/network"
)

func main() {

	errorHandler := func(err error, caller string) {
		fmt.Printf("[ERROR] %s: %s\n", caller, err)
	}

	infoHandler := func(msg string, caller string) {
		fmt.Printf("[INFO] %s: %s\n", caller, msg)
	}

	v4face := network.DefaultV4Interface()

	upnpServer := &upnp.Server{
		ListenAddress: v4face.IP + ":55975",
		SsdpInterface: v4face.Interface,
		ErrorHandler:  errorHandler,
		InfoHandler:   infoHandler,
		Controllers: []upnp.Controller{
			presentation.NewController(),
			contentdirectory.NewServiceController(),
		},
		OnDeviceCreate: func(s *upnp.Server) error {
			infoHandler("call:OnDeviceCreate (time to setup Device)", "app")
			return nil
		},
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		// terminate ssdp server
		infoHandler("gracefully shutting down...", "app")
		upnpServer.Shutdown()
	}()

	infoHandler("server starting", "app")

	_ = upnpServer.ListenAndServe()

	infoHandler("server stopped", "app")
}
