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

	if err := backend.Init("storage/media", "storage/cache"); err != nil {
		slog.Error("PANIC", "err", err)
		return
	}
	scanner := backend.NewScanner()
	scanner.Scan()

	v4face := network.DefaultV4Interface()
	listenAddress := v4face.ListenAddress(55975)

	deviceDescription := &device.Description{
		SpecVersion: device.Version,
		Device: &device.Device{
			DeviceType:   dlnaserver.DeviceType,
			FriendlyName: "SZ",
			UDN:          device.NewUDN(device.DefaultFriendlyName()),
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
			VendorXML: []device.VendorXML{
				device.BuildVendorXML("dlna:X_DLNADOC", "DMS-1.50", "urn:schemas-dlna-org:device-1-0"),
				device.BuildVendorXML("sec:X_ProductCap", "smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec", "http://www.sec.co.kr/dlna"),
			},
		},
		Location: "/rootDesc.xml",
	}

	deviceController := deviceinfo.NewController(deviceDescription)
	cdsController := contentdirectory.NewController()

	dlnaServer := &dlnaserver.Server{
		ListenAddress:     listenAddress,
		SsdpInterface:     v4face.Interface,
		DeviceDescription: deviceDescription,
		Debug:             dlnaserver.DebugFull,
		BeforeHttpStart: func(s *dlnaserver.Server, mux *http.ServeMux, desc *device.Description) {

			mux.HandleFunc("/", s.HookFunc(deviceController.HandlePresentationURL))
			mux.HandleFunc("/rootDesc.xml", s.HookFunc(deviceController.HandleDescRoot))

			mux.HandleFunc("/icons/", s.HookFunc(deviceController.HandleIcons))

			mux.HandleFunc("/ContentDirectory.xml", s.HookFunc(cdsController.HandleSCPDURL))
			mux.HandleFunc("/ctl/{profile}/ContentDirectory", s.HookFunc(cdsController.HandleControlURL))
			mux.HandleFunc("/evt/{profile}/ContentDirectory", s.HookFunc(cdsController.HandleEventSubURL))
			mux.HandleFunc("/thumbs/{profile}/{image}", s.HookFunc(cdsController.HandleThumbnailURL))

		},
	}

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
