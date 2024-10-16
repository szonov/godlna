package main

import (
	"github.com/szonov/godlna/internal/backend"
	"github.com/szonov/godlna/internal/contentdirectory"
	"github.com/szonov/godlna/internal/deviceinfo"
	"github.com/szonov/godlna/internal/dlnaserver"
	"github.com/szonov/godlna/internal/logger"
	"github.com/szonov/godlna/internal/net_utils"
	"github.com/szonov/godlna/upnp/device"
	"github.com/szonov/godlna/upnp/ssdp"
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
		os.Exit(1)
	}

	backend.Scanner.Scan("0")
	//return

	// ------------------------------------------------------------
	// 2. setup device
	// ------------------------------------------------------------

	v4face := net_utils.DefaultV4Interface()
	listenAddress := v4face.ListenAddress(55975)
	friendlyName := "SZ"
	serverHeader := dlnaserver.DefaultServerHeader()

	deviceDescription := &device.Description{
		SpecVersion: device.Version,
		Device: &device.Device{
			DeviceType:   "urn:schemas-upnp-org:device:MediaServer:1",
			FriendlyName: friendlyName,
			UDN:          device.NewUDN(friendlyName),
			Manufacturer: "Home",
			ModelName:    "DLNA Server",
			IconList: []device.Icon{
				{Mimetype: "image/jpeg", Width: 120, Height: 120, Depth: 24, URL: "/device/icons/DeviceIcon120.jpg"},
				{Mimetype: "image/jpeg", Width: 48, Height: 48, Depth: 24, URL: "/device/icons/DeviceIcon48.jpg"},
				{Mimetype: "image/png", Width: 120, Height: 120, Depth: 24, URL: "/device/icons/DeviceIcon120.png"},
				{Mimetype: "image/png", Width: 48, Height: 48, Depth: 24, URL: "/device/icons/DeviceIcon48.png"},
			},
			ServiceList: []*device.Service{
				{
					ServiceType: contentdirectory.ServiceType,
					ServiceId:   contentdirectory.ServiceId,
					SCPDURL:     "/cds/desc.xml",
					ControlURL:  "/cds/{profile}/ctl",
					EventSubURL: "/cds/{profile}/evt",
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
		Location: "/device/desc.xml",
	}

	// ------------------------------------------------------------
	// 3. setup services and handlers
	// ------------------------------------------------------------

	if err := deviceinfo.Init(deviceDescription); err != nil {
		slog.Error("PANIC: device info init", "err", err)
		os.Exit(1)
	}

	if err := contentdirectory.Init(); err != nil {
		slog.Error("PANIC: content directory init", "err", err)
		os.Exit(1)
	}

	// ------------------------------------------------------------
	// 4. setup dlna http server
	// ------------------------------------------------------------

	dlnaServer := &dlnaserver.Server{
		ListenAddress:     listenAddress,
		SsdpInterface:     v4face.Interface,
		DeviceDescription: deviceDescription,
		ServerHeader:      serverHeader,
		OnHttpRequest: func(s *dlnaserver.Server, next http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
			logger.DebugRequest(r, false, false)
			w.Header().Set("Server", s.ServerHeader)
			next.ServeHTTP(w, r)
		},
		BeforeStart: func(s *dlnaserver.Server, mux *http.ServeMux, s_ *ssdp.Server) {
			// index
			mux.HandleFunc("/", s.Hook(deviceinfo.HandlePresentationURL))

			// device
			mux.HandleFunc("/device/desc.xml", s.Hook(deviceinfo.HandleDeviceDescriptionURL))
			mux.HandleFunc("/device/icons/", s.Hook(deviceinfo.HandleIcons))

			// content directory
			mux.HandleFunc("/cds/desc.xml", s.Hook(contentdirectory.HandleSCPDURL))
			mux.HandleFunc("/cds/{profile}/ctl", s.Hook(contentdirectory.HandleControlURL))
			mux.HandleFunc("/cds/{profile}/evt", s.Hook(contentdirectory.HandleEventSubURL))

			// content
			mux.HandleFunc("/t/{path...}", s.Hook(contentdirectory.HandleThumbnailURL))
			mux.HandleFunc("/v/{path...}", s.Hook(contentdirectory.HandleVideoURL))
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
