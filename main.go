package main

import (
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/contentdirectory"
	"github.com/szonov/godlna/internal/deviceinfo"
	"github.com/szonov/godlna/internal/dlnaserver"
	"github.com/szonov/godlna/internal/logger"
	"github.com/szonov/godlna/internal/network"
	"github.com/szonov/godlna/upnp/device"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
)

func main() {

	logger.InitLogger()

	// ------------------------------------------------------------
	// 1. initialize backend
	// ------------------------------------------------------------

	if err := backend.Init("storage/media", "storage/cache"); err != nil {
		slog.Error("PANIC", "err", err)
		return
	}

	scanner := backend.NewScanner()
	scanner.Scan()

	// ------------------------------------------------------------
	// 2. setup device
	// ------------------------------------------------------------

	v4face := network.DefaultV4Interface()
	listenAddress := v4face.ListenAddress(55975)
	friendlyName := "SZ"
	serverHeader := dlnaserver.DefaultServerHeader()

	deviceDescription := &device.Description{
		SpecVersion: device.Version,
		Device: &device.Device{
			DeviceType:   "urn:schemas-upnp-org:device:MediaServer:1",
			FriendlyName: friendlyName,
			UDN:          device.NewUDN(friendlyName + "-01"),
			Manufacturer: "Home",
			ModelName:    "DLNA Server",
			IconList: []device.Icon{
				{Mimetype: "image/jpeg", Width: 120, Height: 120, Depth: 24, URL: "/icons/DeviceIcon120.jpg"},
				{Mimetype: "image/jpeg", Width: 48, Height: 48, Depth: 24, URL: "/icons/DeviceIcon48.jpg"},
				{Mimetype: "image/png", Width: 120, Height: 120, Depth: 24, URL: "/icons/DeviceIcon120.png"},
				{Mimetype: "image/png", Width: 48, Height: 48, Depth: 24, URL: "/icons/DeviceIcon48.png"},
			},
			ServiceList: []*device.Service{
				{
					ServiceType: contentdirectory.ServiceType,
					ServiceId:   contentdirectory.ServiceId,
					SCPDURL:     "/ContentDirectory.xml",
					ControlURL:  "/ctl/{profile}/ContentDirectory",
					EventSubURL: "/evt/{profile}/ContentDirectory",
				},
			},
			PresentationURL: "http://" + listenAddress + "/",
			VendorXML: device.NewVendorXML().
				Add("dlna", "urn:schemas-dlna-org:device-1-0",
					device.VendorValue("X_DLNADOC", "DMS-1.50"),
				).
				Add("sec", "http://www.sec.co.kr/dlna",
					device.VendorValue("ProductCap", "smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec"),
					device.VendorValue("X_ProductCap", "smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec"),
				),
		},
		Location: "/rootDesc.xml",
	}

	// ------------------------------------------------------------
	// 3. setup services and handlers
	// ------------------------------------------------------------

	if err := contentdirectory.Init(); err != nil {
		slog.Error("PANIC: content directory init", "err", err)
		return
	}

	deviceController := deviceinfo.NewController(deviceDescription)

	// ------------------------------------------------------------
	// 4. setup dlna http server
	// ------------------------------------------------------------

	dlnaServer := &dlnaserver.Server{
		ListenAddress:     listenAddress,
		SsdpInterface:     v4face.Interface,
		DeviceDescription: deviceDescription,
		ServerHeader:      serverHeader,
		Debug:             dlnaserver.DebugLight,
		//Debug: dlnaserver.DebugFull,
		BeforeHttpStart: func(s *dlnaserver.Server, mux *http.ServeMux, desc *device.Description) {

			mux.HandleFunc("/", s.HookFunc(deviceController.HandlePresentationURL))
			mux.HandleFunc("/rootDesc.xml", s.HookFunc(deviceController.HandleDescRoot))

			mux.HandleFunc("/icons/", s.HookFunc(deviceController.HandleIcons))

			mux.HandleFunc("/ContentDirectory.xml", s.HookFunc(contentdirectory.HandleSCPDURL))
			mux.HandleFunc("/ctl/{profile}/ContentDirectory", s.HookFunc(contentdirectory.HandleControlURL))
			mux.HandleFunc("/evt/{profile}/ContentDirectory", s.HookFunc(contentdirectory.HandleEventSubURL))
			mux.HandleFunc("/thumbs/{profile}/{image}", s.HookFunc(contentdirectory.HandleThumbnailURL))
			mux.HandleFunc("/video/{profile}/{video}", s.HookFunc(contentdirectory.HandleVideoURL))

		},
	}

	// ------------------------------------------------------------
	// 5. start
	// ------------------------------------------------------------

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		// terminate ssdp server
		slog.Debug("gracefully shutting down...")
		dlnaServer.Shutdown()
	}()

	slog.Info("server starting...")

	_ = dlnaServer.ListenAndServe()

	slog.Info("server stopped")

}
