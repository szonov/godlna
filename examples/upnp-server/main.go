package main

import (
	"fmt"
	"github.com/szonov/go-upnp-lib"
	"github.com/szonov/go-upnp-lib/network"
	"net/http"
	"os"
	"os/signal"
)

type IndexController struct {
	notify upnp.InfoHandlerFunc
}

func (c *IndexController) notifyInfo(msg string) {
	if c.notify != nil {
		c.notify(msg, "INDEX")
	}
}

func (c *IndexController) OnServerStart(s *upnp.Server) error {
	c.notify = s.InfoHandler
	c.notifyInfo("Initialized IndexController")
	return nil
}

func (c *IndexController) Handle(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" {
		c.notifyInfo("route `/` handled")
		_, _ = w.Write([]byte("Index Page"))
		return true
	}
	return false
}

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
			new(IndexController),
		},
		OnDeviceCreate: func(s *upnp.Server) error {
			infoHandler("call:OnDeviceCreate (time to setup Device)", "app")
			return nil
		},
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		// terminate ssdp server
		infoHandler("gracefully shutting down...", "app")
		upnpServer.Shutdown()
	}()

	infoHandler("server starting", "app")

	_ = upnpServer.ListenAndServe()

	infoHandler("server stopped", "app")
}
