package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/szonov/go-upnp-lib/ssdp"
)

func main() {

	errorHandler := func(err error, caller string) {
		fmt.Printf("[ERROR] %s: %s\n", caller, err)
	}

	infoHandler := func(msg string, caller string) {
		fmt.Printf("[INFO] %s: %s\n", caller, msg)
	}

	ssdpServer := &ssdp.Server{
		Location:       "http://192.168.0.100",
		DeviceType:     "urn:schemas-upnp-org:device:MediaServer:1",
		DeviceUUID:     "da2cc462-0000-0000-0000-44fd2452e03f",
		ServiceList:    []string{"urn:schemas-upnp-org:service:ConnectionManager:1"},
		ErrorHandler:   errorHandler,
		InfoHandler:    infoHandler,
		NotifyInterval: 10 * time.Second,
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
