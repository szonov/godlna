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
		fmt.Printf("[%s] ERROR %s\n", caller, err)
	}

	infoHandler := func(msg string, caller string) {
		fmt.Printf("[%s] INFO %s\n", caller, msg)
	}

	ssdpServer := &ssdp.Server{
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

	if err := ssdpServer.ListenAndServe(); err != nil {
		errorHandler(err, "app")
	}

	infoHandler("server stopped", "app")
}
