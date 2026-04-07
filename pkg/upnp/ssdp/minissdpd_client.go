package ssdp

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
)

type MinissdpdClient struct {
	// Socket path, default is  "/var/run/minissdpd.sock"
	Socket string

	// o SSDP options
	o *Options

	quit chan struct{}
}

func NewMinissdpdClient(o *Options, socket ...string) *MinissdpdClient {
	if len(socket) == 0 {
		socket = append(socket, "/var/run/minissdpd.sock")
	}
	return &MinissdpdClient{
		Socket: socket[0],
		o:      o,
	}
}

func (s *MinissdpdClient) validate() error {
	if !IsSocket(s.Socket) {
		return fmt.Errorf("%s is not a valid socket", s.Socket)
	}
	return s.o.Validate()
}

func (s *MinissdpdClient) Start() error {
	if err := s.validate(); err != nil {
		return err
	}
	s.quit = make(chan struct{})
	runPeriodic(s.sendAlive, s.o.NotifyInterval, s.quit)
	return nil
}

func (s *MinissdpdClient) Stop() error {
	close(s.quit)
	s.sendByeBye()
	return nil
}

func (s *MinissdpdClient) udpNotify(messageFn func(string) []byte) {

	conn, err := net.Dial("udp", MulticastAddrPort)
	if err != nil {
		slog.Error("can not connect to", "address", MulticastAddrPort, "err", err)
		return
	}

	defer func(conn net.Conn) {
		if err = conn.Close(); err != nil {
			slog.Error("can not close connect to", "address", MulticastAddrPort, "err", err)
		}
	}(conn)

	for _, target := range s.o.AllTargets() {
		msg := messageFn(target)
		if len(msg) == 0 {
			return
		}
		if _, err = conn.Write(msg); err != nil {
			slog.Error("can not send message to", "address", MulticastAddrPort, "err", err)
		}
	}
}

func (s *MinissdpdClient) minissdpdNotify() error {
	var err error

	var minissdpd net.Conn
	if minissdpd, err = net.Dial("unix", s.Socket); err != nil {
		return err
	}
	defer func(minissdpd net.Conn) {
		err = minissdpd.Close()
		if err != nil {
			slog.Warn("error closing minissdpd", "err", err)
		}
	}(minissdpd)

	for _, target := range s.o.AllTargets() {
		buf := &bytes.Buffer{}
		usn := s.o.UsnFromTarget(target)

		_, err = fmt.Fprintf(buf, "\x04%c%s%c%s%c%s%c%s",
			len(target), target,
			len(usn), usn,
			len(s.o.ServerHeader), s.o.ServerHeader,
			len(s.o.Location), s.o.Location,
		)
		if err != nil {
			return err
		}

		if _, err = minissdpd.Write(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

// sendAlive submit to minissdpd and write Alive message to udp connection
func (s *MinissdpdClient) sendAlive() {
	if err := s.minissdpdNotify(); err != nil {
		slog.Error("failed to submit to minissdpd", "err", err)
	}
	s.udpNotify(func(target string) []byte { return s.o.AliveMessage(target) })
}

// sendByeBye write Bye message to udp connection
func (s *MinissdpdClient) sendByeBye() {
	s.udpNotify(func(target string) []byte { return s.o.ByeByeMessage(target) })
}
