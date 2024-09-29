package main

import (
	"fmt"
	"github.com/szonov/go-upnp-lib/network"
	"github.com/szonov/go-upnp-lib/ssdp"
	"os"
	"os/signal"
)

func main() {

	errorHandler := func(err error, caller string) {
		fmt.Printf("[ERROR] %s: %s\n", caller, err)
	}

	infoHandler := func(msg string, caller string) {
		fmt.Printf("[INFO] %s: %s\n", caller, msg)
	}

	v4face := network.DefaultV4Interface()

	ssdpServer := &ssdp.Server{
		Location:     "http://" + v4face.IP + "/device.xml",
		DeviceType:   "urn:schemas-upnp-org:device:MediaServer:1",
		DeviceUDN:    "uuid:da2cc462-0000-0000-0000-44fd2452e03f",
		ServiceList:  []string{"urn:schemas-upnp-org:service:ConnectionManager:1"},
		Interface:    v4face.Interface,
		ErrorHandler: errorHandler,
		InfoHandler:  infoHandler,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		// terminate ssdp server
		infoHandler("gracefully shutting down...", "app")
		ssdpServer.Shutdown()
	}()

	infoHandler("server starting", "app")

	_ = ssdpServer.ListenAndServe()

	infoHandler("server stopped", "app")
}
