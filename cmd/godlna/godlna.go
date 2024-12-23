package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/szonov/godlna/dlna"
	"github.com/szonov/godlna/indexer"
	"github.com/szonov/godlna/logger"
	"github.com/szonov/godlna/network"
	"github.com/szonov/godlna/pkg/upnp/ssdp"
	"github.com/yookoala/realpath"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	dsn             = "database=godlna"
	videoDirectory  = ""
	friendlyName    = "GoDLNA"
	listenInterface = ""
	listenIP        = ""
	listenPort      = 50003
)

func main() {
	v4faceDefault := network.DefaultV4Interface()

	flag.StringVar(&dsn, "dsn", "database=godlna", "database `dsn` string")
	flag.StringVar(&videoDirectory, "root", defaultVideoRoot(), "`directory` containing video files")
	flag.StringVar(&friendlyName, "name", friendlyName, "`friendlyName` as you see it on TV")
	flag.StringVar(&listenInterface, "eth", v4faceDefault.Interface.Name, "network `interface` name")
	flag.StringVar(&listenIP, "ip", v4faceDefault.IP, "on which `ip` run dlna server")
	flag.IntVar(&listenPort, "port", listenPort, "on which `port` run dlna server")
	flag.Parse()

	logger.InitLogger(slog.LevelDebug)

	videoDirectory = validateRootDirectory(videoDirectory)
	psql := makeDbConnection(dsn)
	idx := makeIndexer(videoDirectory, psql)
	v4face, listenAddress := makeNetwork(listenInterface, listenIP)
	dlnaServer := makeDLNAServer(videoDirectory, friendlyName, listenAddress, psql)
	ssdpServer := makeSsdpServer(dlnaServer, v4face.Interface)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		// terminate indexer, ssdp, dlna servers
		slog.Debug("gracefully shutting down...")

		idx.Stop()
		ssdpServer.Shutdown()
		dlnaServer.Shutdown()
	}()

	idx.Start()
	ssdpServer.Start()
	_ = dlnaServer.ListenAndServe()
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

func validateRootDirectory(videoDirectory string) string {
	var err error

	if videoDirectory, err = filepath.Abs(videoDirectory); err != nil {
		criticalError(err)
	}

	if videoDirectory, err = realpath.Realpath(videoDirectory); err != nil {
		criticalError(err)
	}

	if !indexer.FileExists(videoDirectory) {
		criticalError(fmt.Errorf("video directory does not exist: %s", videoDirectory))
	}

	return videoDirectory
}

func makeIndexer(videoDirectory string, psql *pgxpool.Pool) *indexer.Indexer {
	if !indexer.FFMpegBinPathAutodetect() {
		criticalError(fmt.Errorf("ffmpeg not detected"))
	}

	if !indexer.FFProbeBinPathAutodetect() {
		criticalError(fmt.Errorf("ffprobe not detected"))
	}
	return indexer.NewIndexer(videoDirectory, psql)
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

func makeDLNAServer(root string, friendlyName string, listenAddress string, psql *pgxpool.Pool) *dlna.Server {
	srv := dlna.NewServer(root, friendlyName, listenAddress, psql)
	srv.DebugRequest = true
	return srv
}

func makeSsdpServer(s *dlna.Server, iface *net.Interface) *ssdp.Server {
	services := make([]string, 0)
	for _, serv := range s.DeviceDescription.Device.ServiceList {
		services = append(services, serv.ServiceType)
	}

	return &ssdp.Server{
		Location:     "http://" + s.ListenAddress + s.DeviceDescription.Location,
		ServerHeader: dlna.ServerHeader,
		DeviceType:   s.DeviceDescription.Device.DeviceType,
		DeviceUDN:    s.DeviceDescription.Device.UDN,
		ServiceList:  services,
		Interface:    iface,
	}
}

func defaultVideoRoot() string {
	if indexer.FileExists("/volume1/video") {
		return "/volume1/video"
	}
	if indexer.FileExists("/volume2/video") {
		return "/volume2/video"
	}
	return "./"
}

func criticalError(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}
