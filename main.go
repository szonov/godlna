package main

import (
	"flag"
	"fmt"
	"github.com/szonov/godlna/internal/config"
	"github.com/szonov/godlna/internal/contentdirectory"
	"github.com/szonov/godlna/internal/deviceinfo"
	"github.com/szonov/godlna/internal/dlnaserver"
	"github.com/szonov/godlna/internal/ffmpeg"
	"github.com/szonov/godlna/internal/logger"
	"github.com/szonov/godlna/internal/net_utils"
	"github.com/szonov/godlna/internal/store"
	"github.com/szonov/godlna/upnp/device"
	"github.com/szonov/godlna/upnp/ssdp"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"time"
)

func main() {
	// ------------------------------------------------------------
	// read configuration from file,
	// passed in command line argument `-config config.toml`
	// ------------------------------------------------------------

	var configFile string
	flag.StringVar(&configFile, "config", "config.toml", "Config file")
	flag.Parse()

	var cfg *config.Config
	var err error

	if cfg, err = config.Read(configFile); err != nil {
		criticalError(err)
	}

	logger.InitLogger(cfg.Logger.Level)

	// ------------------------------------------------------------
	// initialize store
	// ------------------------------------------------------------

	if cfg.Store.MediaDir == "" || cfg.Store.CacheDir == "" {
		criticalError(fmt.Errorf("missing configuration for media dir or cache dir"))
	}
	if cfg.Store.CacheLifeTime == 0 {
		cfg.Store.CacheLifeTime = 10 * time.Minute
	}
	err = store.Init(cfg.Store.MediaDir, cfg.Store.CacheDir, cfg.Store.CacheLifeTime)
	if err != nil {
		criticalError(err)
	}

	if cfg.Programs.FFMpeg == "" {
		cfg.Programs.FFMpeg = "ffmpeg"
	}
	if cfg.Programs.FFProbe == "" {
		cfg.Programs.FFProbe = "ffprobe"
	}
	ffmpeg.SetFFMpegBinPath(cfg.Programs.FFMpeg)
	ffmpeg.SetFFProbeBinPath(cfg.Programs.FFProbe)

	// ------------------------------------------------------------
	// setup device
	// ------------------------------------------------------------

	v4face := net_utils.DefaultV4Interface(cfg.Network.IFace, cfg.Network.IP)
	listenAddress := v4face.ListenAddress(cfg.Server.Port)

	var friendlyName, udn, serverHeader string
	if friendlyName = cfg.Device.FriendlyName; friendlyName == "" {
		friendlyName = "Video"
	}
	if udn = cfg.Device.UUID; udn == "" {
		udn = device.NewUDN(friendlyName)
	}
	if serverHeader = cfg.Server.Header; serverHeader == "" {
		serverHeader = dlnaserver.DefaultServerHeader()
	}

	deviceDescription := &device.Description{
		SpecVersion: device.Version,
		Device: &device.Device{
			DeviceType:   "urn:schemas-upnp-org:device:MediaServer:1",
			FriendlyName: friendlyName,
			UDN:          udn,
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
					ControlURL:  "/cds/ctl",
					EventSubURL: "/cds/evt",
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
	// setup services and handlers
	// ------------------------------------------------------------

	if err = deviceinfo.Init(deviceDescription); err != nil {
		criticalError(err)
	}
	if err = contentdirectory.Init(); err != nil {
		criticalError(err)
	}

	// ------------------------------------------------------------
	// setup dlna http server
	// ------------------------------------------------------------

	var ssdpInterface *net.Interface
	if !cfg.Ssdp.Disable {
		ssdpInterface = v4face.Interface
	}
	var requestID int64 = 0
	dlnaServer := &dlnaserver.Server{
		ListenAddress:     listenAddress,
		SsdpInterface:     ssdpInterface,
		DeviceDescription: deviceDescription,
		ServerHeader:      serverHeader,
		OnHttpRequest: func(s *dlnaserver.Server, next http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Request-ID", strconv.FormatInt(atomic.AddInt64(&requestID, 1), 10))
			logger.DebugRequest(r, true, false)
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
			mux.HandleFunc("/cds/ctl", s.Hook(contentdirectory.HandleControlURL))
			mux.HandleFunc("/cds/evt", s.Hook(contentdirectory.HandleEventSubURL))

			// content
			mux.HandleFunc("/t/{path...}", s.Hook(contentdirectory.HandleThumbnailURL))
			mux.HandleFunc("/v/{path...}", s.Hook(contentdirectory.HandleVideoURL))
		},
	}

	// ------------------------------------------------------------
	// debug used configuration
	// ------------------------------------------------------------

	slog.Debug("---------------------------------------------")
	slog.Debug("CFG", "Friendly Name", friendlyName)
	slog.Debug("CFG", "UDN", udn)
	slog.Debug("CFG", "Listen Address", listenAddress)
	slog.Debug("CFG", "SSDP Enabled", ssdpInterface != nil)
	slog.Debug("CFG", "Media Dir", cfg.Store.MediaDir)
	slog.Debug("CFG", "Cache Dir", cfg.Store.CacheDir)
	slog.Debug("CFG", "Cache Life Time", cfg.Store.CacheLifeTime)
	slog.Debug("CFG", "ffprobe", cfg.Programs.FFProbe)
	slog.Debug("CFG", "ffmpeg", cfg.Programs.FFMpeg)
	slog.Debug("---------------------------------------------")

	// ------------------------------------------------------------
	// start
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

func criticalError(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}
