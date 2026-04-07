package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/szonov/godlna/dlna"
	"github.com/szonov/godlna/dlna/backend"
	"github.com/szonov/godlna/logger"
	"github.com/szonov/godlna/network"
	"github.com/szonov/godlna/pkg/ffmpeg"
	"github.com/szonov/godlna/pkg/ffprobe"
	"github.com/szonov/godlna/pkg/upnp/ssdp"
)

type StringList []string

func (s *StringList) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *StringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	dsn             string
	videoDirs       StringList
	friendlyName    string
	listenInterface string
	listenIP        string
	listenPort      int
	minissdpdSocket string
	logLevel        string
)

func main() {
	v4faceDefault := network.DefaultV4Interface()

	flag.StringVar(&dsn, "dsn", "database=godlna", "database `dsn` string")
	flag.Var(&videoDirs, "root", "`directory` containing video files, can be specified multiple times. (default is "+defaultVideoRoot()+")")
	flag.StringVar(&friendlyName, "name", "GoDLNA", "`friendlyName` as you see it on TV")
	flag.StringVar(&listenInterface, "eth", v4faceDefault.Interface.Name, "network `interface` name")
	flag.StringVar(&listenIP, "ip", v4faceDefault.IP, "on which `ip` run dlna server")
	flag.IntVar(&listenPort, "port", 50003, "on which `port` run dlna server")
	flag.StringVar(&minissdpdSocket, "minissdpd", defaultMinissdpd(), "Minissdp `socket` file, pass empty string to disable")
	flag.StringVar(&logLevel, "log", "info", "Log `level`, accepted values are: debug, info, warn, error")
	flag.Parse()

	makeLogger(logLevel)

	if len(videoDirs) == 0 {
		videoDirs = append(videoDirs, defaultVideoRoot())
	}

	psql := makeDbConnection(dsn)
	back := makeBackend(videoDirs, psql)

	if minissdpdSocket != "" && !ssdp.IsSocket(minissdpdSocket) {
		slog.Warn("minissdpd socket disabled, incorrect", "socket", minissdpdSocket)
		minissdpdSocket = ""
	}

	v4face, listenAddress := makeNetwork(listenInterface, listenIP)
	dlnaServer := makeDLNAServer(friendlyName, listenAddress, back)

	var ssdpServer ssdp.Server
	var ssdpInfo []any
	ssdpOpts := makeSsdpOptions(dlnaServer)
	if minissdpdSocket != "" {
		ssdpServer = ssdp.NewMinissdpdClient(ssdpOpts, minissdpdSocket)
		ssdpInfo = append(ssdpInfo, "socket", minissdpdSocket)
	} else {
		ssdpServer = ssdp.NewUdpServer(ssdpOpts, v4face.Interface)
		ssdpInfo = append(ssdpInfo, "address", ssdp.MulticastAddrPort, "if", v4face.Interface.Name)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGKILL,
		syscall.SIGTERM,
	)

	go func() {
		<-c
		// terminate backend, ssdp, dlna servers
		slog.Debug("gracefully shutting down...")

		slog.Info("stopping DLNA backend")
		if err := back.Stop(); err != nil {
			slog.Error(err.Error())
		}

		slog.Info("stopping SSDP server")
		if err := ssdpServer.Stop(); err != nil {
			slog.Error(err.Error())
		}

		slog.Info("stopping DLNA server")
		dlnaServer.Shutdown()
	}()

	slog.Info("starting DLNA backend")
	if err := back.Start(); err != nil {
		criticalError(err)
	}
	slog.Info("starting SSDP server", ssdpInfo...)
	go startSsdpServer(ssdpServer)

	slog.Info("starting DLNA server", "address", dlnaServer.ListenAddress)
	if err := dlnaServer.ListenAndServe(); err != nil {
		slog.Error(err.Error())
	}
}

func makeLogger(level string) {
	var loggerLogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		loggerLogLevel = slog.LevelDebug
	case "info":
		loggerLogLevel = slog.LevelInfo
	case "warn":
		loggerLogLevel = slog.LevelWarn
	case "error":
		loggerLogLevel = slog.LevelError
	default:
		criticalError(fmt.Errorf("invalid log level: %s", logLevel))
	}
	logger.InitLogger(loggerLogLevel)
}

func makeDbConnection(dsn string) *pgxpool.Pool {

	var config *pgxpool.Config
	var engine *pgxpool.Pool
	var err error

	if config, err = pgxpool.ParseConfig(dsn); err != nil {
		criticalError(err)
	}
	if engine, err = pgxpool.NewWithConfig(context.Background(), config); err != nil {
		criticalError(err)
	}
	return engine
}

func makeBackend(dirs []string, psql *pgxpool.Pool) *backend.Backend {

	if !ffmpeg.Autodetect() {
		criticalError(fmt.Errorf("ffmpeg binary not found"))
	}
	if !ffprobe.Autodetect() {
		criticalError(fmt.Errorf("ffprobe binary not found"))
	}
	driver := backend.NewPostgresDriver(psql)
	back, err := backend.NewBackend(dirs, driver)
	if err != nil {
		criticalError(err)
	}
	return back
}

func makeNetwork(eth string, ip string) (network.V4Interface, string) {
	v4face := network.DefaultV4Interface(eth, ip)
	if !v4face.Valid() {
		criticalError(fmt.Errorf("network interface is not valid: %s / %s", eth, ip))
	}
	addr := v4face.ListenAddress(listenPort)
	if addr == "" {
		criticalError(fmt.Errorf("could not find listen address: %s / %d", v4face, listenPort))
	}
	return v4face, addr
}

func makeDLNAServer(friendlyName string, listenAddress string, back *backend.Backend) *dlna.Server {
	srv := dlna.NewServer(friendlyName, listenAddress, back)
	srv.DebugRequest = true
	//srv.DebugRequestHeader = true
	//srv.DebugRequestBody = true
	return srv
}

func makeSsdpOptions(s *dlna.Server) *ssdp.Options {
	services := make([]string, 0)
	for _, serv := range s.DeviceDescription.Device.ServiceList {
		services = append(services, serv.ServiceType)
	}
	return &ssdp.Options{
		Location:     "http://" + s.ListenAddress + s.DeviceDescription.Location,
		ServerHeader: dlna.ServerHeader,
		DeviceType:   s.DeviceDescription.Device.DeviceType,
		DeviceUDN:    s.DeviceDescription.Device.UDN,
		ServiceList:  services,
	}
}

func startSsdpServer(srv ssdp.Server) {
	if err := srv.Start(); err != nil {
		criticalError(err)
	}
}

func defaultVideoRoot() string {
	if _, err := os.Stat("/volume1/video"); err == nil {
		return "/volume1/video"
	}
	if _, err := os.Stat("/volume2/video"); err == nil {
		return "/volume2/video"
	}
	if d, err := filepath.Abs("./"); err == nil {
		return d
	}
	return "./"
}

func defaultMinissdpd() string {
	socket := "/var/run/minissdpd.sock"
	if ssdp.IsSocket(socket) {
		return socket
	}
	return ""
}

func criticalError(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}
