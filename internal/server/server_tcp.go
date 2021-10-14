package server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
)

type ServerTCP struct {
	listener net.Listener
	config   *ServerConfig
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

func (srv *ServerTCP) Start() {
	protoName := "TCP"

	if srv.config.proto == ProtoTLS {
		protoName = "TLS"
		msg := fmt.Sprintf("Creating %s server on port %d\n", protoName, srv.config.port)
		srv.config.event.Send("runtime", srv.config.name, msg)

		cer, err := tls.LoadX509KeyPair(
			srv.config.certPem, srv.config.certKey,
		)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}
		portStr := fmt.Sprintf(":%d", srv.config.port)

		ln, err := tls.Listen("tcp", portStr, tlsConfig)
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		srv.listener = ln

	} else {
		msg := fmt.Sprintf("Creating %s server on port %d\n", protoName, srv.config.port)
		srv.config.event.Send("runtime", srv.config.name, msg)

		portStr := fmt.Sprintf(":%d", srv.config.port)
		ln, err := net.Listen("tcp", portStr)
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()
		srv.listener = ln
	}

	msg := fmt.Sprintf("Starting %s server on port %d\n", protoName, srv.config.port)
	srv.config.event.Send("runtime", srv.config.name, msg)
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println("Accepted...calling handler...")
		go srv.connHandler(conn)
	}
}

func (srv *ServerTCP) connHandler(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		srv.config.event.Send("request", srv.config.name, msg)
		if srv.config.hcServer {
			srv.config.metric.Inc("requests_hc")
		} else {
			srv.config.metric.Inc("requests_service")
		}

		healthy := fmt.Sprintf("%s\n", srv.config.hc.GetHealthyStr())
		n, err := conn.Write([]byte(healthy))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}
