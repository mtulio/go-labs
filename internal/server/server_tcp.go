package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type ServerTCP struct {
	listener net.Listener
	lnConfig *net.ListenConfig
	config   *ServerConfig
	quit     chan interface{}
}

func NewTCPServer(cfg *ServerConfig) (*ServerTCP, error) {
	log.SetFlags(log.Lshortfile)

	srv := ServerTCP{
		config: cfg,
	}

	srv.config.event.Send(
		"runtime", cfg.name, "Server TCP Created",
	)
	return &srv, nil
}

// Start is responsible to setup the TCP server, listen,
// and accept new connections routing the connections to
// the handler with non-blocking allowing parallel connections.
func (srv *ServerTCP) Start() {
	protoName := "TCP"
	srv.lnConfig = &net.ListenConfig{
		Control:   TCPControl,
		KeepAlive: -1,
	}
	if srv.config.proto == ProtoTLS {
		protoName = "TLS"
		srv.sendEvent(fmt.Sprintf("Creating %s server on port %d\n", protoName, srv.config.port))

		cer, err := tls.LoadX509KeyPair(
			srv.config.certPem, srv.config.certKey,
		)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		portStr := fmt.Sprintf(":%d", srv.config.port)

		ln, err := tls.Listen("tcp", portStr, tlsConfig)
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		srv.listener = ln

	} else {
		srv.sendEvent(fmt.Sprintf("Creating %s server on port %d\n", protoName, srv.config.port))

		portStr := fmt.Sprintf(":%d", srv.config.port)
		ln, err := srv.lnConfig.Listen(context.Background(), "tcp", portStr)
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		srv.listener = ln
	}

	srv.sendEvent(fmt.Sprintf("Starting %s server on port %d\n", protoName, srv.config.port))
	srv.quit = make(chan interface{})
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			select {
			case <-srv.quit:
				srv.sendEvent("TCP Server detected Unhealthy state: stopping TCP Listener\n")
				return
			default:
				log.Println("TCP server: Accept error: ", err)
			}
			continue
		}

		// Avoid to call connection handler when HC start to fail
		if !srv.config.hc.GetHealthy() {
			continue
		}
		go func() {
			if srv.config.debug {
				srv.sendEvent("TCP Connection accepted, calling handler.")
			}
			srv.connHandler(conn)
		}()
	}
}

func (srv *ServerTCP) Stop() {
	srv.listener.Close()
}

func (srv *ServerTCP) connHandler(conn net.Conn) {
	defer conn.Close()
	for {
		netMsg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			// log.Printf("Error ReadString: [%v]", err)
			switch err {
			case io.EOF:
				return
			default:
				log.Printf("Error ReadString: [%v]", err)
			}
			return
		}

		srv.config.event.Send("request", srv.config.name, netMsg)
		if srv.config.hcServer {
			srv.config.metric.Inc("requests_hc")
		} else {
			srv.config.metric.Inc("requests_service")
		}

		cmd := strings.TrimSpace(string(netMsg))
		if cmd == "STOP" {
			break
		}

		n, err := conn.Write([]byte(string(srv.config.hc.GetHealthyStr())))
		if err != nil {
			log.Println("Error writing response: ", n, err)
		}
		if srv.config.debug {
			log.Printf("received from %v: [%s]", conn.RemoteAddr(), cmd)
		}
	}
}

// StartController is a infinity loop to watch the Health Check
// Controller and force the server to not answer TCP requests when
// the health check should be in failing state.
func (srv *ServerTCP) StartController() {
	waitReconcile := time.Duration(1 * time.Second)
	waitStateTransiction := time.Duration(5 * time.Second)

	srv.sendEvent("TCP Server controller: starting in 5 seconds.")
	srv.ControllerWaiter(waitStateTransiction)
	for {
		// Use the controller only in health check servers
		if !(srv.config.hcServer) {
			fmt.Println("Ignoring Server Controller, it is enabled only in Health check servers.")
			return
		}

		// State> health check is failing and server is up.
		// Action: Server needs to be stopped
		if !(srv.config.hc.GetHealthy()) && (srv.ServerPortIsOpen()) {
			srv.sendEvent("TCP Server controller: unhealthy state detected, closing the TCP listener and waiting for transiction...")
			close(srv.quit)
			srv.Stop()
			srv.ControllerWaiter(waitStateTransiction)
			continue
		}

		// State> health check has cleaned and server is down.
		// Action: Server needs to be started
		if (srv.config.hc.GetHealthy()) && !(srv.ServerPortIsOpen()) {
			srv.sendEvent("TCP Server controller: healthy state detected, starting TCP listener server...")
			go srv.Start()
			srv.ControllerWaiter(waitStateTransiction)
			continue
		}

		// State> Health check failing and server is down.
		// Action: register the event and wait to to reconcile period
		if !(srv.config.hc.GetHealthy()) && !(srv.ServerPortIsOpen()) {
			srv.sendEvent("TCP Server controller: health check is failing and server is stopped, waiting to state change")
		}

		// Last state> Health check and server listening successfully.
		// Action: wait to to reconcile period.
		srv.ControllerWaiter(waitReconcile)
	}
}

// ControllerWaiter wait in seconds
func (srv *ServerTCP) ControllerWaiter(t time.Duration) {
	time.Sleep(t)
}

// ServerPortIsOpen checks whether the TCP port is Opened, and return
// a boolean. True when the TCP Port is opened.
func (srv *ServerTCP) ServerPortIsOpen() bool {
	isOpen := true
	_, err := net.Dial("tcp", fmt.Sprintf(":%d", srv.config.port))
	if err != nil {
		isOpen = false
	}
	return isOpen
}

// TCPControl set of flags of TCP listener to reuse the
// socket when the service needs to be restored.
// This function will work only for unix systems. To port it
// to other systems, we advise to use the external lib go-reuseport.
// https://github.com/libp2p/go-reuseport/blob/master/control_unix.go
func TCPControl(network, address string, c syscall.RawConn) error {
	var err error
	c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			return
		}
	})
	return err
}

func (srv *ServerTCP) sendEvent(msg string) {
	srv.config.event.Send("runtime", srv.config.name, msg)
}
